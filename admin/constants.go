package admin

import "gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"

var (
	defaultPage     = 1
	defaultPageSize = 10
	tokenLength     = 32
)

var querier = dbsqlc.New()
