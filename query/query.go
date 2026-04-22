package query

type Query struct {
	Select      string
	Update_Data any
	Havings     []string
	Where       any
	Wheres      []Where
	Order       string
}
