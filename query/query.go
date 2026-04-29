package query

type Query struct {
	Select     string
	UpdateData any
	Havings    []string
	Where      any
	Wheres     []Where
	Order      string
}
