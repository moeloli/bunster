package generator

import (
	"fmt"

	"github.com/yassinebenaid/bunster/ast"
	"github.com/yassinebenaid/bunster/ir"
)

func (g *generator) handleFunction(buf *InstructionBuffer, function *ast.Function) {
	var body InstructionBuffer

	g.handleBreakPoints(buf, function.BreakPoints)

	g.handleRedirections(&body, function.Redirections)

	for _, cmd := range function.Body {
		g.generate(&body, cmd)
	}

	buf.add(ir.Function{Name: function.Name, Body: body, Subshell: function.SubShell})
}

func (g *generator) handleExit(buf *InstructionBuffer, exit ast.Exit) {
	buf.add(ir.Exit{Code: g.handleExpression(buf, exit.Code)})
}

func (g *generator) handleReturn(buf *InstructionBuffer, b *ast.Return) {
	buf.add(ir.Set{Name: fmt.Sprintf("breakpoint%d", b.BreakPoint), Value: ir.Literal("true")})

	buf.add(ir.Literal("return;"))
}

func (g *generator) handleBreak(buf *InstructionBuffer, b *ast.Break) {
	buf.add(ir.Set{Name: fmt.Sprintf("breakpoint%d", b.BreakPoint), Value: ir.Literal("true")})

	switch b.Type {
	case ast.RETURN:
		buf.add(ir.Literal("return;"))
	default:
		buf.add(ir.Literal("break;"))
	}
}

func (g *generator) handleContinue(buf *InstructionBuffer, b *ast.Continue) {
	buf.add(ir.Set{Name: fmt.Sprintf("breakpoint%d", b.BreakPoint), Value: ir.Literal("true")})

	switch b.Type {
	case ast.RETURN:
		buf.add(ir.Literal("return;"))
	default:
		buf.add(ir.Literal("continue;"))
	}
}

func (g *generator) handleWait(buf *InstructionBuffer, _ ast.Wait) {
	buf.add(ir.Literal("shell.WaitGroup.Wait()\n"))
}

func (g *generator) handleEmbed(_ *InstructionBuffer, embed ast.Embed) {
	g.embeds = append(g.embeds, embed...)
}
