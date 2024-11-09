package courseload

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

func loadPage(offset, size int) []Course {
	res, err := http.Get(fmt.Sprintf("https://wapi.unh.edu/dhub/api/courses/all/202410?page[offset]=%d&page[limit]=%d", offset, size))

	if err != nil {
		return nil
	}

	defer res.Body.Close()

	var rawData map[string]interface{}

	json.NewDecoder(res.Body).Decode(&rawData)

	var courses []Course

	for _, course := range rawData["data"].([]interface{}) {
		var c Course

		str, _ := json.Marshal(course)
		json.Unmarshal(str, &c)

		courses = append(courses, c)
	}

	return courses
}

func loadTotalCountAndBuckets(bucketSize int) [][]int {
	res, err := http.Get("https://wapi.unh.edu/dhub/api/courses/all/202410?page[offset]=0&page[limit]=1")

	if err != nil {
		return nil
	}

	defer res.Body.Close()

	var rawData map[string]interface{}

	json.NewDecoder(res.Body).Decode(&rawData)

	var totalCount int = int(rawData["total-count"].(float64))

	var buckets [][]int

	for i := 0; i < totalCount; i += bucketSize {
		if i+bucketSize > totalCount {
			bucketSize = totalCount - i
		}

		buckets = append(buckets, []int{i, bucketSize})
	}

	return buckets
}

func LoadCourses() []Course {
	var courses []Course
	var buckets [][]int = loadTotalCountAndBuckets(64)
	var waitgroup sync.WaitGroup

	for _, bucket := range buckets {
		waitgroup.Add(1)
		go func(offset, size int) {
			defer waitgroup.Done()
			courses = append(courses, loadPage(offset, size)...)
		}(bucket[0], bucket[1])
	}

	waitgroup.Wait()

	return courses
}
