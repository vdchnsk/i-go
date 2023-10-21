package parser

import (
	"fmt"
	"strconv"

	"github.com/vdchnsk/i-go/src/ast"
	"github.com/vdchnsk/i-go/src/lexer"
	"github.com/vdchnsk/i-go/src/token"
)

const (
	_ int = iota
	LOWEST
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	CALL
)

type Parser struct {
	lexer *lexer.Lexer

	currToken token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn

	errors []string
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:  l,
		errors: []string{},
	}

	// call NextToken twice to have currToken and peekToken both set
	p.NextToken()
	p.NextToken()

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifer)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExression)

	return p
}

var precedences = map[token.TokenType]int{
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
	token.LPAREN:   CALL,
}

func (p *Parser) peekPrecedence() int {
	if precedence, ok := precedences[p.peekToken.Type]; ok {
		return precedence
	}
	return LOWEST
}

func (p *Parser) currPrecedence() int {
	if precedence, ok := precedences[p.currToken.Type]; ok {
		return precedence
	}
	return LOWEST
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) PeekError(t token.TokenType) {
	msg := fmt.Sprintf(
		"expected next token to be %s, got %s instead",
		t, p.peekToken.Type,
	)
	p.errors = append(p.errors, msg)
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) NextToken() {
	p.currToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.currTokenIs(token.EOF) {
		statement := p.parseStatement()

		program.Statements = append(program.Statements, statement)

		p.NextToken()
	}

	return program
}

func (p *Parser) parseIdentifer() ast.Expression {
	return &ast.Identifier{
		Token: p.currToken,
		Value: p.currToken.Literal,
	}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	prefExp := &ast.PrefixExpression{
		Token:    p.currToken,
		Operator: p.currToken.Literal,
	}

	p.NextToken()

	prefExp.Right = p.parseExpression(PREFIX)

	return prefExp
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	infExp := &ast.InfixExpression{
		Token:    p.currToken,
		Operator: p.currToken.Literal,
		Left:     left,
	}

	precedence := p.currPrecedence()
	p.NextToken()
	infExp.Right = p.parseExpression(precedence)

	return infExp
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	intLit := &ast.IntegerLiteral{Token: p.currToken}

	value, err := strconv.ParseInt(p.currToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.currToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	intLit.Value = value

	return intLit
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.currToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	statement := &ast.ExpressionStatement{Token: p.currToken}

	statement.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.NextToken()
	}

	return statement
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.currToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.currToken.Type)
		return nil
	}

	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.NextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	statement := &ast.LetStatement{Token: p.currToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	statement.Identifier = &ast.Identifier{Token: p.currToken, Value: p.currToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.NextToken()
	statement.Value = p.parseExpression(LOWEST)

	for p.peekTokenIs(token.SEMICOLON) {
		p.NextToken()
	}

	return statement
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	statement := &ast.ReturnStatement{Token: p.currToken}

	p.NextToken()

	statement.Value = p.parseExpression(LOWEST)

	for p.peekTokenIs(token.SEMICOLON) {
		p.NextToken()
	}

	return statement
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{
		Token: p.currToken,
		Value: p.currTokenIs(token.TRUE),
	}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.NextToken()

	expression := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return expression
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.currToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.NextToken()

	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.NextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	blockStatement := &ast.BlockStatement{Token: p.currToken}
	blockStatement.Statements = []ast.Statement{}

	p.NextToken()

	endOfBlockToken := token.RBRACE
	for !p.currTokenIs(token.TokenType(endOfBlockToken)) && !p.currTokenIs(token.EOF) {
		statement := p.parseStatement()

		blockStatement.Statements = append(blockStatement.Statements, statement)

		p.NextToken()
	}
	return blockStatement
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	funcLit := &ast.FuncLiteral{
		Token: p.currToken,
	}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	funcLit.Parameters = p.ParseFuncParams()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	funcLit.Body = *p.parseBlockStatement()

	return funcLit
}

func (p *Parser) ParseFuncParams() []*ast.Identifier {
	params := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.NextToken()
		return params
	}

	p.NextToken()

	parameter := &ast.Identifier{
		Token: p.currToken,
		Value: p.currToken.Literal,
	}
	params = append(params, parameter)

	for p.peekTokenIs(token.COMMA) {
		p.NextToken()
		p.NextToken()
		parameter := &ast.Identifier{
			Token: p.currToken,
			Value: p.currToken.Literal,
		}
		params = append(params, parameter)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return params
}

func (p *Parser) parseCallExression(fn ast.Expression) ast.Expression {
	callExpr := &ast.CallExpression{
		Token:    p.currToken,
		Function: fn,
	}
	callExpr.Argments = p.parseCallExressionArgs()
	return callExpr
}

func (p *Parser) parseCallExressionArgs() []ast.Expression {
	args := []ast.Expression{}

	if p.peekTokenIs(token.RPAREN) {
		p.NextToken()
		return args
	}

	p.NextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.NextToken()
		p.NextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) currTokenIs(expectedCurrentToken token.TokenType) bool {
	return p.currToken.Type == expectedCurrentToken
}

func (p *Parser) peekTokenIs(expectedPeekToken token.TokenType) bool {
	return p.peekToken.Type == expectedPeekToken
}

func (p *Parser) expectPeek(expectedToken token.TokenType) bool {
	if p.peekTokenIs(expectedToken) {
		p.NextToken()
		return true
	}

	p.PeekError(expectedToken)
	return false
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)