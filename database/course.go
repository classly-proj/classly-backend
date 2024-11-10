package database

import (
	"database/sql"
	"fmt"
	"time"

	"hacknhbackend.eparker.dev/courseload"
	"hacknhbackend.eparker.dev/util"
)

func CourseUpdates() {
	start := time.Now()
	courses := courseload.LoadCourses()
	util.Log.Basic(fmt.Sprintf("Loaded %d courses in %v", len(courses), time.Since(start)))

	crns, err := GetCourseCRNs()

	if err != nil {
		util.Log.Error(fmt.Sprintf("Error getting course CRNS: %v\n\n> Will Override", err))

		for _, course := range courses {
			InsertCourse(course)
		}

		return
	}

	// Map to map[string]bool
	var crnsMap map[string]int = make(map[string]int)

	for _, course := range courses {
		crnsMap[course.CRN] = 1
	}

	var inserts, deletes int = 0, 0

	for _, crn := range crns {
		if _, ok := crnsMap[crn]; !ok {
			DeleteCourse(crn)
			deletes++
		} else {
			crnsMap[crn] = 2
		}
	}

	for _, course := range courses {
		if crnsMap[course.CRN] == 1 {
			InsertCourse(course)
			inserts++
		}
	}

	util.Log.Status(fmt.Sprintf("Inserted %d courses, deleted %d courses", inserts, deletes))
}

func InsertCourse(course courseload.Course) error {
	err := QueuedExec(INSERT_COURSE_STATEMENT, course.CRN, course.Data.Title, course.Data.Subject, course.Data.Number, course.Data.SectionNum, course.Data.Description)
	if err != nil {
		return err
	}

	for _, instructor := range course.Data.Instructors {
		err := QueuedExec(INSERT_INSTUCTOR_STATEMENT, instructor.LastName, instructor.FirstName, instructor.Email, course.CRN)
		if err != nil {
			return err
		}
	}

	for _, meeting := range course.Data.Meetings {
		err := QueuedExec(INSERT_MEETING_STATEMENT, meeting.Days, meeting.Building, meeting.Room, meeting.Time, course.CRN)
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteCourse(term_crn string) error {
	err := QueuedExec("DELETE FROM courses WHERE term_crn = ?;", term_crn)
	if err != nil {
		return err
	}

	err = QueuedExec("DELETE FROM instructors WHERE term_crn = ?;", term_crn)
	if err != nil {
		return err
	}

	err = QueuedExec("DELETE FROM meetings WHERE term_crn = ?;", term_crn)
	if err != nil {
		return err
	}

	return nil
}

func GetCourse(term_crn string) (*courseload.Course, error) {
	row := QueuedQueryRow(SELECT_COUSE_STATEMENT, term_crn)

	var title, subject_code, course_number, section_number, description string
	err := row.Scan(&term_crn, &title, &subject_code, &course_number, &section_number, &description)
	if err != nil {
		return nil, err
	}

	instructors := make([]courseload.Instructor, 0)
	rows, err := QueuedQuery(SELECT_INSTRUCTORS_STATEMENT, term_crn)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var id int
		var last_name, first_name, email string
		err = rows.Scan(&id, &last_name, &first_name, &email)
		if err != nil {
			return nil, err
		}

		instructors = append(instructors, courseload.Instructor{
			LastName:  last_name,
			FirstName: first_name,
			Email:     email,
		})
	}

	meetings := make([]courseload.Meeting, 0)
	rows, err = QueuedQuery(SELECT_MEETINGS_STATEMENT, term_crn)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var id int
		var days, building, room, time string
		err = rows.Scan(&id, &days, &building, &room, &time)
		if err != nil {
			return nil, err
		}

		meetings = append(meetings, courseload.Meeting{
			Days:     days,
			Building: building,
			Room:     room,
			Time:     time,
		})
	}

	return &courseload.Course{
		CRN: term_crn,
		Data: courseload.CourseData{
			Title:       title,
			Subject:     subject_code,
			Number:      course_number,
			Description: description,
			Instructors: instructors,
			Meetings:    meetings,
			SectionNum:  section_number,
		},
	}, nil
}

func GetCourseCRNs() ([]string, error) {
	rows, err := QueuedQuery("SELECT term_crn FROM courses;")
	if err != nil {
		return nil, err
	}

	courses := make([]string, 0)

	for rows.Next() {
		var term_crn string
		err = rows.Scan(&term_crn)
		if err != nil {
			return nil, err
		}

		courses = append(courses, term_crn)
	}

	return courses, nil
}

var QueryableKeys = map[string]string{
	"term_crn":       "CRN",
	"title":          "Title",
	"subject_code":   "Subject",
	"course_number":  "Number",
	"subject-number": "Subject & Number",
}

func QueryCourse(key string, values ...string) ([]courseload.Course, error) {
	if _, ok := QueryableKeys[key]; !ok {
		return nil, fmt.Errorf("key %s is not queryable", key)
	}

	var rows *sql.Rows
	var err error

	switch key {
	case "title":
		rows, err = QueuedQuery("SELECT term_crn FROM courses WHERE title LIKE ?", "%"+values[0]+"%")
	case "subject-number":
		rows, err = QueuedQuery("SELECT term_crn FROM courses WHERE subject_code = ? AND course_number LIKE ?", values[0], "%"+values[1]+"%")
	default:
		rows, err = QueuedQuery("SELECT term_crn FROM courses WHERE "+key+" = ?", values[0])
	}

	if err != nil {
		return nil, err
	}

	courses := make([]courseload.Course, 0)

	for rows.Next() {
		var term_crn string
		err = rows.Scan(&term_crn)
		if err != nil {
			return nil, err
		}

		course, err := GetCourse(term_crn)
		if err != nil {
			return nil, err
		}

		courses = append(courses, *course)
	}

	return courses, nil
}
