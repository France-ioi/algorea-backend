package database

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type withClause struct {
	expressions []clause.Expression
}

func (m *withClause) Build(builder clause.Builder) {
	_, err := builder.WriteString("WITH ")
	mustNotBeError(err)
	for i, expr := range m.expressions {
		if i != 0 {
			_, err = builder.WriteString(", ")
			mustNotBeError(err)
		}
		expr.Build(builder)
	}
}

func (m *withClause) ModifyStatement(statement *gorm.Statement) {
	selectClause := statement.Clauses["SELECT"]
	if beforeExpression := selectClause.BeforeExpression; beforeExpression != nil {
		if previousWithClause, ok := beforeExpression.(*withClause); ok {
			previousWithClause.expressions = append(previousWithClause.expressions, m.expressions...)
			return
		}
	}
	selectClause.BeforeExpression = m
	statement.Clauses["SELECT"] = selectClause
}

var (
	_ clause.Expression      = &withClause{}
	_ gorm.StatementModifier = &withClause{}
)
