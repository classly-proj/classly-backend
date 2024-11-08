package directions

type Floor struct {
	Number     int
	Hallways   []*Hallway
	Stairwells []*Stairwell
}

type Hallway struct {
	Number           int
	AdjacentHallways map[int]*Hallway
	Rooms            []*Room
	Stairwells       []*Stairwell
}

type Room struct {
	Number int
}

type Stairwell struct {
	Number int
	Floors map[int]*Floor
}
