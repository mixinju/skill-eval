package eval

// Unit 一个评测的最小集合
type Unit struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Input   string `json:"input"`
	Success bool   `json:"success"`
}
