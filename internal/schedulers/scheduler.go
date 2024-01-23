package schedulers

type Scheduler interface {
	Apply(int) ([]string, error)
	Restore([]string)
	serialize() *string
}
