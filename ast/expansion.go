package ast

type Param struct {
	Name  string
	Index Expression
}

type ParameterExpansion struct {
	Name  string
	Index Expression
}

type VarOrDefault struct {
	Parameter    Param
	Index        Expression
	Default      Expression
	CheckForNull bool
}

type VarOrSet struct {
	Parameter Param
	Index     Expression
	Default   Expression
}

type VarOrFail struct {
	Parameter Param
	Index     Expression
	Error     Expression
}

type CheckAndUse struct {
	Parameter Param
	Index     Expression
	Value     Expression
}

type ChangeCase struct {
	Parameter Param
	Index     Expression
	Operator  string
	Pattern   Expression
}

type VarCount struct {
	Parameter Param
	Index     Expression
}

type MatchAndRemove struct {
	Parameter Param
	Index     Expression
	Operator  string
	Pattern   Expression
}

type MatchAndReplace struct {
	Parameter Param
	Index     Expression
	Operator  string
	Pattern   Expression
	Value     Expression
}

type Transform struct {
	Parameter Param
	Index     Expression
	Operator  string
}

type Slice struct {
	Parameter Param
	Index     Expression
	Offset    Arithmetic
	Length    Arithmetic
}

func (ParameterExpansion) node() {}
func (VarOrDefault) node()       {}
func (VarOrSet) node()           {}
func (VarOrFail) node()          {}
func (CheckAndUse) node()        {}
func (ChangeCase) node()         {}
func (VarCount) node()           {}
func (MatchAndRemove) node()     {}
func (MatchAndReplace) node()    {}
func (Transform) node()          {}
func (Slice) node()              {}

// Expressions
func (ParameterExpansion) expr() {}
func (VarOrDefault) expr()       {}
func (VarOrSet) expr()           {}
func (VarOrFail) expr()          {}
func (CheckAndUse) expr()        {}
func (ChangeCase) expr()         {}
func (VarCount) expr()           {}
func (MatchAndRemove) expr()     {}
func (MatchAndReplace) expr()    {}
func (Transform) expr()          {}
func (Slice) expr()              {}

func (v ParameterExpansion) string() string { return string(v.Name) }
func (VarOrDefault) string() string         { return "" }
func (VarOrSet) string() string             { return "" }
func (VarOrFail) string() string            { return "" }
func (CheckAndUse) string() string          { return "" }
func (ChangeCase) string() string           { return "" }
func (VarCount) string() string             { return "" }
func (MatchAndRemove) string() string       { return "" }
func (MatchAndReplace) string() string      { return "" }
func (Transform) string() string            { return "" }
func (Slice) string() string                { return "" }
