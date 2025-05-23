package parser

import (
	"github.com/yassinebenaid/bunster/ast"
	"github.com/yassinebenaid/bunster/token"
)

func (p *parser) parseTestCommand() ast.Statement {
	p.proceed()
	for p.curr.Type == token.BLANK || p.curr.Type == token.NEWLINE {
		p.proceed()
	}
	if p.curr.Type == token.DOUBLE_RIGHT_BRACKET {
		p.error("expected a conditional expression before `]]`")
		return nil
	}
	expr := p.parseTestExpression(false)

	if expr == nil {
		p.error("bad conditional expression, unexpected token `%s`", p.curr)
		return nil
	}
	for p.curr.Type == token.BLANK || p.curr.Type == token.NEWLINE {
		p.proceed()
	}

	if p.curr.Type != token.DOUBLE_RIGHT_BRACKET {
		p.error("expected `]]` to close conditional expression, found `%s`", p.curr)
		return nil
	}
	p.proceed()

	test := ast.Test{Expr: expr}

loop:
	for {
		switch {
		case p.curr.Type == token.BLANK:
			p.proceed()
		case p.isRedirectionToken():
			p.HandleRedirection(&test.Redirections)
		default:
			break loop
		}
	}

	if p.curr.Type == token.HASH {
		for p.curr.Type != token.NEWLINE && p.curr.Type != token.EOF {
			p.proceed()
		}
	}

	if !p.isControlToken() && p.curr.Type != token.EOF {
		p.error("unexpected token `%s`", p.curr)
		return nil
	}

	return test
}

func (p *parser) parsePosixTestCommand() ast.Statement {
	testKeyword := p.curr.Type == token.TEST

	p.proceed()
	if p.curr.Type == token.BLANK {
		p.proceed()
	}
	if p.curr.Type == token.RIGHT_BRACKET {
		p.error("expected a conditional expression before `]`")
		return nil
	}
	expr := p.parsePosixTestExpression(false)
	if expr == nil {
		p.error("bad conditional expression, unexpected token `%s`", p.curr)
		return nil
	}

	if !testKeyword {
		if p.curr.Type != token.RIGHT_BRACKET {
			p.error("expected `]` to close conditional expression, found `%s`", p.curr)
			return nil
		}
		p.proceed()
	}

	test := ast.Test{Expr: expr}

loop:
	for {
		switch {
		case p.curr.Type == token.BLANK:
			p.proceed()
		case p.isRedirectionToken():
			p.HandleRedirection(&test.Redirections)
		default:
			break loop
		}
	}

	if p.curr.Type == token.HASH {
		for p.curr.Type != token.NEWLINE && p.curr.Type != token.EOF {
			p.proceed()
		}
	}

	if !p.isControlToken() && p.curr.Type != token.EOF {
		p.error("bad conditional expression, unexpected token `%s`", p.curr)
		return nil
	}

	return test
}

func (p *parser) parseTestExpression(prefix bool) ast.Expression {
	var expr ast.Expression
	if p.curr.Type == token.EXCLAMATION {
		p.proceed()
		if p.curr.Type == token.BLANK {
			p.proceed()
		}
		var neg ast.Negation
		if p.curr.Type != token.DOUBLE_RIGHT_BRACKET {
			neg.Operand = p.parseTestExpression(true)
		}
		if neg.Operand == nil {
			p.error("bad conditional expression, unexpected token `%s`", p.curr)
			return nil
		}
		expr = neg
	} else if p.curr.Type == token.LEFT_PAREN {
		p.proceed()
		if p.curr.Type == token.BLANK {
			p.proceed()
		}
		if p.curr.Type != token.DOUBLE_RIGHT_BRACKET {
			expr = p.parseTestExpression(false)
		}
		if expr == nil {
			p.error("bad conditional expression, unexpected token `%s`", p.curr)
			return nil
		}
		if p.curr.Type != token.RIGHT_PAREN {
			p.error("expected a closing `)`, found `%s`", p.curr)
			return nil
		}
		p.proceed()
		if p.curr.Type == token.BLANK {
			p.proceed()
		}
	} else {
		expr = p.parseConditionals()
	}

	for !prefix && (p.curr.Type == token.AND || p.curr.Type == token.OR) {
		operator := p.curr.Literal
		p.proceed()
		for p.curr.Type == token.BLANK || p.curr.Type == token.NEWLINE {
			p.proceed()
		}

		bin := ast.Binary{Left: expr, Operator: operator}
		if p.curr.Type != token.DOUBLE_RIGHT_BRACKET {
			bin.Right = p.parseTestExpression(true)
		}
		if bin.Right == nil {
			p.error("bad conditional expression, unexpected token `%s`", p.curr)
			return nil
		}
		expr = bin
	}

	return expr
}

func (p *parser) parsePosixTestExpression(prefix bool) ast.Expression {
	var expr ast.Expression
	if p.curr.Type == token.EXCLAMATION {
		p.proceed()
		if p.curr.Type == token.BLANK {
			p.proceed()
		}
		var neg ast.Negation
		if p.curr.Type != token.RIGHT_BRACKET {
			neg.Operand = p.parsePosixTestExpression(true)
		}
		if neg.Operand == nil {
			p.error("bad conditional expression, unexpected token `%s`", p.curr)
			return nil
		}
		expr = neg
	} else if p.curr.Type == token.LEFT_PAREN {
		p.proceed()
		if p.curr.Type == token.BLANK {
			p.proceed()
		}
		if p.curr.Type != token.RIGHT_BRACKET {
			expr = p.parsePosixTestExpression(false)
		}
		if expr == nil {
			p.error("bad conditional expression, unexpected token `%s`", p.curr)
			return nil
		}
		if p.curr.Type != token.RIGHT_PAREN {
			p.error("expected a closing `)`, found `%s`", p.curr)
			return nil
		}
		p.proceed()
		if p.curr.Type == token.BLANK {
			p.proceed()
		}
	} else {
		expr = p.parseConditionals()
	}

	for !prefix {
		operator := p.parsePosixConditionalBinaryOperator()
		if operator == "" {
			break
		}
		if operator == "-a" {
			operator = "&&"
		} else {
			operator = "||"
		}

		bin := ast.Binary{Left: expr, Operator: operator}
		if p.curr.Type != token.RIGHT_BRACKET && p.curr.Type != token.DOUBLE_RIGHT_BRACKET {
			bin.Right = p.parsePosixTestExpression(true)
		}
		if bin.Right == nil {
			p.error("bad conditional expression, unexpected token `%s`", p.curr)
			return nil
		}
		expr = bin
	}

	return expr
}

func (p *parser) parseConditionals() ast.Expression {
	if exp := p.parseUnaryConditional(); exp != nil {
		if p.curr.Type == token.BLANK {
			p.proceed()
		}
		return exp
	}

	exp := p.parseExpression()

	if p.curr.Type == token.BLANK {
		p.proceed()
	}

	operator := p.parseConditionalBinaryOperator()
	if operator == "" {
		return exp
	}

	bin := ast.Binary{Left: exp, Operator: operator}

	if p.curr.Type != token.DOUBLE_RIGHT_BRACKET && p.curr.Type != token.RIGHT_BRACKET {
		if operator == "=~" {
			bin.Right = p.parsePatternExpression()
		} else {
			bin.Right = p.parseExpression()
		}
	}

	if bin.Right == nil {
		p.error("bad conditional expression, expected an operand after `%s`, found `%s`", operator, p.curr)
		return nil
	}

	if p.curr.Type == token.BLANK {
		p.proceed()
	}

	return bin
}

func (p *parser) parseUnaryConditional() ast.Expression {
	if p.curr.Type == token.MINUS {
		switch p.next.Literal {
		case "a", "b", "c", "d", "e", "f", "g", "h", "k", "p", "r", "s",
			"t", "u", "w", "x", "G", "L", "N", "O", "S", "z", "n", "v":
			if p.next2.Type != token.BLANK {
				break
			}

			u := ast.Unary{
				Operator: "-" + p.next.Literal,
			}
			p.proceed()
			p.proceed()
			p.proceed()

			if p.curr.Type != token.DOUBLE_RIGHT_BRACKET && p.curr.Type != token.RIGHT_BRACKET {
				u.Operand = p.parseExpression()
			}
			if u.Operand == nil {
				p.error("bad conditional expression, expected an operand after %s, found `%s`", u.Operator, p.curr)
				return nil
			}
			return u
		}
	}

	return nil
}

func (p *parser) parseConditionalBinaryOperator() string {
	switch p.curr.Type {
	case token.ASSIGN, token.EQ, token.NOT_EQ, token.GT, token.LT, token.EQ_TILDE:
		if p.next.Type != token.BLANK {
			break
		}

		operator := p.curr.Literal
		p.proceed()
		p.proceed()

		return operator
	case token.MINUS:
		switch p.next.Literal {
		case "ef", "nt", "ot", "eq", "ne", "lt", "le", "gt", "ge":
			if p.next2.Type != token.BLANK {
				break
			}

			p.proceed()
			operator := "-" + p.curr.Literal
			p.proceed()
			p.proceed()

			return operator
		}
	}

	return ""
}

func (p *parser) parsePosixConditionalBinaryOperator() string {
	if p.curr.Type == token.MINUS {
		switch p.next.Literal {
		case "a", "o":
			if p.next2.Type != token.BLANK {
				break
			}

			p.proceed()
			operator := "-" + p.curr.Literal
			p.proceed()
			p.proceed()

			return operator
		}
	}
	return ""
}

func (p *parser) parsePatternExpression() ast.Expression {
	var exprs []ast.Expression
	// TODO: handle special variables in test commands

loop:
	for {
		switch p.curr.Type {
		case token.BLANK, token.NEWLINE, token.EOF:
			break loop
		case token.SIMPLE_EXPANSION:
			exprs = append(exprs, ast.Var(p.curr.Literal))
		case token.SINGLE_QUOTE:
			exprs = append(exprs, p.parseLiteralString())
		case token.DOUBLE_QUOTE:
			exprs = append(exprs, p.parseString())
		case token.DOLLAR_PAREN:
			exprs = append(exprs, p.parseCommandSubstitution())
		case token.GT_PAREN, token.LT_PAREN:
			exprs = append(exprs, p.parseProcessSubstitution())
		case token.DOLLAR_BRACE:
			exprs = append(exprs, p.parseParameterExpansion())
		case token.DOLLAR_DOUBLE_PAREN:
			exprs = append(exprs, p.parseArithmeticSubstitution())
		default:
			exprs = append(exprs, ast.Word(p.curr.Literal))
		}

		p.proceed()
	}

	return concat(exprs, false)
}
