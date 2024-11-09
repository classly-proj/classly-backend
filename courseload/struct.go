package courseload

type Instructor struct {
	LAST_NAME  string `json:"LAST_NAME"`
	FIRST_NAME string `json:"FIRST_NAME"`
	EMAIL      string `json:"EMAIL"`
}

type Meeting struct {
	DAYS     string `json:"DAYS"`
	BUILDING string `json:"BUILDING"`
	ROOM     string `json:"ROOM"`
	TIME     string `json:"TIME"`
}

type CourseData struct {
	SYVSCHD_CRSE_LONG_TITLE string       `json:"SYVSCHD_CRSE_LONG_TITLE"`
	SYVSCHD_SUBJ_CODE       string       `json:"SYVSCHD_SUBJ_CODE"`
	SYVSCHD_CRSE_NUMB       string       `json:"SYVSCHD_CRSE_NUMB"`
	SYVSCHD_CRSE_DESC       string       `json:"SYVSCHD_CRSE_DESC"`
	INSTRUCTORS             []Instructor `json:"INSTRUCTORS"`
	MEETINGS                []Meeting    `json:"MEETINGS"`
}

type Course struct {
	TERM_CRN    string     `json:"TERM_CRN"`
	COURSE_DATA CourseData `json:"COURSE_DATA"`
}
