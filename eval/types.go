package eval

// Unit 一个评测的最小集合
type Unit struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Input   string `json:"input"`
	Success bool   `json:"success"` // 是否运行成功
}

// Report 一个流程的评测报告数据
type Report struct {
}
