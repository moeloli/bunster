package generator

import (
	"fmt"

	"github.com/yassinebenaid/bunster/ast"
	"github.com/yassinebenaid/bunster/ir"
)

func (g *generator) handleParameterExpansionVarLength(buf *InstructionBuffer, expression ast.VarLength) ir.Instruction {
	switch v := expression.Parameter.(type) {
	case ast.SpecialVar:
		return ir.VarLength{Name: string(v), Special: true}
	case ast.PositionalSpread:
		return ir.ReadSpecialVar("#")
	case ast.Var:
		return ir.VarLength{Name: string(v)}
	case ast.ArrayAccess:
		return ir.VarLength{Name: string(v.Name), Index: ir.ParseInt{Value: g.handleExpression(buf, v.Index)}}
	}
	panic("unknown parameter kind")
}

func (g *generator) handleParameterExpansionVarOrDefault(buf *InstructionBuffer, expression ast.VarOrDefault) ir.Instruction {
	name := fmt.Sprintf("expr%d", g.expressionsCount)
	buf.add(ir.Declare{Name: name, Value: ir.String("")})

	var def ir.Instruction = ir.String("")
	if expression.Default != nil {
		def = g.handleExpression(buf, expression.Default)
	}

	_if := ir.If{
		Body:      []ir.Instruction{ir.Set{Name: name, Value: g.handleParameter(buf, expression.Parameter)}},
		Alternate: []ir.Instruction{ir.Set{Name: name, Value: def}},
	}

	if expression.UnsetOnly {
		switch v := expression.Parameter.(type) {
		case ast.SpecialVar:
			_if.Condition = ir.TestVarIsSet{Name: ir.Literal(v), Positional: true}
		case ast.Var:
			_if.Condition = ir.TestVarIsSet{Name: ir.String(v)}
		case ast.ArrayAccess:
			_if.Condition = ir.TestVarIsSet{Name: ir.String(v.Name), Index: ir.ParseInt{Value: g.handleExpression(buf, v.Index)}}
		}
	} else {
		_if.Condition = ir.TestAgainsStringLength{String: g.handleParameter(buf, expression.Parameter)}
	}

	buf.add(_if)
	return ir.Literal(name)
}

func (g *generator) handleParameterExpansionVarOrSet(buf *InstructionBuffer, expression ast.VarOrSet) ir.Instruction {
	var def ir.Instruction = ir.String("")
	if expression.Default != nil {
		def = g.handleExpression(buf, expression.Default)
	}

	_if := ir.If{Not: true}
	switch v := expression.Parameter.(type) {
	case ast.Var:
		_if.Body = append(_if.Body, ir.SetVar{Key: string(v), Value: def})
	case ast.ArrayAccess:
		_if.Body = append(_if.Body, ir.SetVar{
			Key:   v.Name,
			Index: ir.ParseInt{Value: g.handleExpression(buf, v.Index)},
			Value: def,
		})
	}

	if expression.UnsetOnly {
		switch v := expression.Parameter.(type) {
		case ast.Var:
			_if.Condition = ir.TestVarIsSet{Name: ir.String(v)}
		case ast.ArrayAccess:
			_if.Condition = ir.TestVarIsSet{Name: ir.String(v.Name), Index: ir.ParseInt{Value: g.handleExpression(buf, v.Index)}}
		}
	} else {
		_if.Condition = ir.TestAgainsStringLength{String: g.handleParameter(buf, expression.Parameter)}
	}

	buf.add(_if)

	switch v := expression.Parameter.(type) {
	case ast.Var:
		return ir.ReadVar(v)
	case ast.ArrayAccess:
		return ir.ReadArrayVar{Name: v.Name, Index: ir.ParseInt{Value: g.handleExpression(buf, v.Index)}}
	default:
		panic("unhandled case")
	}
}

func (g *generator) handleParameterExpansionCheckAndUse(buf *InstructionBuffer, expression ast.CheckAndUse) ir.Instruction {
	name := fmt.Sprintf("expr%d", g.expressionsCount)
	buf.add(ir.Declare{Name: name, Value: ir.String("")})

	var value ir.Instruction = ir.String("")
	if expression.Value != nil {
		value = g.handleExpression(buf, expression.Value)
	}

	_if := ir.If{
		Body: []ir.Instruction{
			ir.Set{Name: name, Value: value},
		},
	}

	if expression.UnsetOnly {
		switch v := expression.Parameter.(type) {
		case ast.SpecialVar:
			_if.Condition = ir.TestVarIsSet{Name: ir.Literal(v), Positional: true}
		case ast.Var:
			_if.Condition = ir.TestVarIsSet{Name: ir.String(v)}
		case ast.ArrayAccess:
			_if.Condition = ir.TestVarIsSet{Name: ir.String(v.Name), Index: ir.ParseInt{Value: g.handleExpression(buf, v.Index)}}
		}
	} else {
		_if.Condition = ir.TestAgainsStringLength{String: g.handleParameter(buf, expression.Parameter)}
	}

	buf.add(_if)
	return ir.Literal(name)
}

func (g *generator) handleParameterExpansionSlice(buf *InstructionBuffer, expression ast.Slice) ir.Instruction {
	offset := fmt.Sprintf("offset%d", g.expressionsCount)
	length := fmt.Sprintf("length%d", g.expressionsCount)

	buf.add(ir.Declare{Name: offset, Value: ir.Literal("0")})
	buf.add(ir.Declare{Name: length, Value: ir.Literal("int(^uint32(0))")})

	for _, arithmetic := range expression.Offset {
		buf.add(ir.Set{Name: offset, Value: g.handleArithmeticExpression(buf, arithmetic)})
	}

	for _, arithmetic := range expression.Length {
		buf.add(ir.Set{Name: length, Value: g.handleArithmeticExpression(buf, arithmetic)})
	}

	return ir.Substring{
		String: g.handleParameter(buf, expression.Parameter),
		Offset: ir.Literal(offset),
		Length: ir.Literal(length),
	}
}

func (g *generator) handleParameterExpansionChangeCase(buf *InstructionBuffer, expression ast.ChangeCase) ir.Instruction {
	var pattern ir.Instruction = ir.String("?")
	if expression.Pattern != nil {
		pattern = g.handleExpression(buf, expression.Pattern)
	}

	if expression.Operator == "^" || expression.Operator == "^^" {
		return ir.StringToUpperCase{
			String:  g.handleParameter(buf, expression.Parameter),
			Pattern: pattern,
			All:     expression.Operator == "^^",
		}
	}

	return ir.StringToLowerCase{
		String:  g.handleParameter(buf, expression.Parameter),
		Pattern: pattern,
		All:     expression.Operator == ",,",
	}
}

func (g *generator) handleParameterExpansionMatchAndRemove(buf *InstructionBuffer, expression ast.MatchAndRemove) ir.Instruction {
	var pattern ir.Instruction = ir.String("")
	if expression.Pattern != nil {
		pattern = g.handleExpression(buf, expression.Pattern)
	}

	if expression.Operator == "#" || expression.Operator == "##" {
		return ir.RemoveMatchingPrefix{
			String:  g.handleParameter(buf, expression.Parameter),
			Pattern: pattern,
			Longest: expression.Operator == "##",
		}
	}

	return ir.RemoveMatchingSuffix{
		String:  g.handleParameter(buf, expression.Parameter),
		Pattern: pattern,
		Longest: expression.Operator == "%%",
	}
}

func (g *generator) handleParameterExpansionMatchAndReplace(buf *InstructionBuffer, expression ast.MatchAndReplace) ir.Instruction {
	var pattern ir.Instruction = ir.String("")
	if expression.Pattern != nil {
		pattern = g.handleExpression(buf, expression.Pattern)
	}

	var repl ir.Instruction = ir.String("")
	if expression.Value != nil {
		repl = g.handleExpression(buf, expression.Value)
	}

	switch expression.Operator {
	case "/", "//":
		return ir.ReplaceMatching{
			String:  g.handleParameter(buf, expression.Parameter),
			Pattern: pattern,
			Value:   repl,
			All:     expression.Operator == "//",
		}
	case "/#":
		return ir.ReplaceMatchingPrefix{
			String:  g.handleParameter(buf, expression.Parameter),
			Pattern: pattern,
			Value:   repl,
		}
	case "/%":
		return ir.ReplaceMatchingSuffix{
			String:  g.handleParameter(buf, expression.Parameter),
			Pattern: pattern,
			Value:   repl,
		}
	}

	return nil
}

func (g *generator) handleParameter(buf *InstructionBuffer, param ast.Parameter) ir.Instruction {
	switch v := param.(type) {
	case ast.SpecialVar:
		return ir.ReadSpecialVar(v)
	case ast.PositionalSpread:
		return ir.ReadSpecialVar("@")
	case ast.Var:
		return ir.ReadVar(v)
	case ast.ArrayAccess:
		return ir.ReadArrayVar{
			Name:  v.Name,
			Index: ir.ParseInt{Value: g.handleExpression(buf, v.Index)},
		}
	}

	panic("unknown parameter kind")
}
