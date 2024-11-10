package courseload

import "encoding/json"

type Instructor struct {
	LastName  string `json:"LAST_NAME"`
	FirstName string `json:"FIRST_NAME"`
	Email     string `json:"EMAIL"`
}

type Meeting struct {
	Days     string `json:"DAYS"`
	Building string `json:"BUILDING"`
	Room     string `json:"ROOM"`
	Time     string `json:"TIME"`
}

type CourseData struct {
	Title       string       `json:"SYVSCHD_CRSE_LONG_TITLE"`
	Subject     string       `json:"SYVSCHD_SUBJ_CODE"`
	Number      string       `json:"SYVSCHD_CRSE_NUMB"`
	Description string       `json:"SYVSCHD_CRSE_DESC"`
	Instructors []Instructor `json:"INSTRUCTORS"`
	Meetings    []Meeting    `json:"MEETINGS"`
	SectionNum  string       `json:"SYVSCHD_SEQ_NUMB"`
}

type Course struct {
	CRN  string     `json:"TERM_CRN"`
	Data CourseData `json:"COURSE_DATA"`
}

func (c *Course) JSON() []byte {
	b, _ := json.Marshal(c)
	return b
}
