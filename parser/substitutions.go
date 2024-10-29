package parser

import (
	"github.com/yassinebenaid/bunny/ast"
	"github.com/yassinebenaid/bunny/token"
)

func (p *Parser) parseCommandSubstitution() ast.Expression {
	var cmds ast.CommandSubstitution
	p.proceed()
	for p.curr.Type == token.BLANK || p.curr.Type == token.NEWLINE {
		p.proceed()
	}

	for p.curr.Type != token.RIGHT_PAREN && p.curr.Type != token.EOF {
		cmdList := p.parseCommandList()
		if cmdList == nil {
			return nil
		}
		cmds = append(cmds, cmdList)
		if p.curr.Type == token.SEMICOLON || p.curr.Type == token.AMPERSAND {
			p.proceed()
		}
		for p.curr.Type == token.BLANK || p.curr.Type == token.NEWLINE {
			p.proceed()
		}
	}

	if len(cmds) == 0 {
		p.error("expeceted a command list after `$(`")
	}

	if p.curr.Type != token.RIGHT_PAREN {
		p.error("unexpected end of file, expeceted `)`")
	}

	return cmds
}
