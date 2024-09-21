package parser

import (
	"github.com/yassinebenaid/nbs/ast"
	"github.com/yassinebenaid/nbs/token"
)

func (*Parser) isRedirectionToken(t token.Token) bool {
	switch t.Type {
	case token.GT, token.DOUBLE_GT:
		return true
	default:
		return false
	}
}

func (p *Parser) HandleRedirection(cmd *ast.Command) {
	switch p.curr.Type {
	case token.GT, token.DOUBLE_GT:
		p.fromStdoutToFile(cmd)
	}
}

func (p *Parser) getRedirectionParser(tt token.TokenType) func(*ast.Command) {
	switch tt {
	case token.GT, token.DOUBLE_GT:
		return p.fromStdoutToFile
	case token.AMPERSAND_GT:
		return p.parseStdoutAndStderrToFileRedirection
	case token.LT, token.TRIPLE_LT:
		return p.parseStdinFromFileRedirection
	case token.GT_AMPERSAND:
		return p.parseStdoutToFileDescriptorRedirection
	case token.FILE_DESCRIPTOR:
		return p.parseFileDescriptorRedirection
	default:
		return nil
	}
}

func (p *Parser) fromStdoutToFile(cmd *ast.Command) {
	var r ast.Redirection
	r.Src = ast.FileDescriptor("1")
	r.Method = p.curr.Literal

	p.proceed()
	if p.curr.Type == token.BLANK {
		p.proceed()
	}

	r.Dst = p.parseField()
	cmd.Redirections = append(cmd.Redirections, r)
}

func (p *Parser) parseStdoutAndStderrToFileRedirection(cmd *ast.Command) {
	var r ast.Redirection
	r.Src = ast.StdoutStderr{}
	r.Method = p.curr.Literal

	p.proceed()
	if p.curr.Type == token.BLANK {
		p.proceed()
	}

	r.Dst = p.parseField()
	cmd.Redirections = append(cmd.Redirections, r)
}

func (p *Parser) parseStdoutToFileDescriptorRedirection(cmd *ast.Command) {
	var r ast.Redirection
	r.Src = ast.FileDescriptor("1")
	r.Method = p.curr.Literal

	p.proceed()
	r.Dst = ast.FileDescriptor(p.curr.Literal)
	p.proceed()

	cmd.Redirections = append(cmd.Redirections, r)
}

func (p *Parser) parseFileDescriptorRedirection(cmd *ast.Command) {
	var r ast.Redirection
	r.Src = ast.FileDescriptor(p.curr.Literal)

	p.proceed()
	r.Method = p.curr.Literal
	p.proceed()

	if p.curr.Type == token.BLANK {
		p.proceed()
	}

	if p.curr.Type == token.FILE_DESCRIPTOR {
		r.Dst = ast.FileDescriptor(p.curr.Literal)
		p.proceed()
	} else {
		r.Dst = p.parseField()
	}

	cmd.Redirections = append(cmd.Redirections, r)
}

func (p *Parser) parseStdinFromFileRedirection(cmd *ast.Command) {
	var r ast.Redirection
	r.Src = ast.FileDescriptor("0")
	r.Method = p.curr.Literal

	p.proceed()
	if p.curr.Type == token.BLANK {
		p.proceed()
	}

	r.Dst = p.parseField()
	cmd.Redirections = append(cmd.Redirections, r)
}
