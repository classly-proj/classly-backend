package database

import (
	"fmt"
	"time"

	"hacknhbackend.eparker.dev/courseload"
)

func CourseUpdates() {
	start := time.Now()
	courses := courseload.LoadCourses()
	fmt.Printf("Loaded %d courses in %v\n", len(courses), time.Since(start))

	for _, course := range courses {
		InsertCourse(course)
	}
}
