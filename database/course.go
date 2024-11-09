package database

import (
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
