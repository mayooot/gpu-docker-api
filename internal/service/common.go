package service

var (
	cpRFPOption = "cp -rf -p %s/* %s/"
)

type copyTask struct {
	Resource    string
	OldResource string
	NewResource string
}
