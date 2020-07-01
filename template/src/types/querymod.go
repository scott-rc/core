package types

import "github.com/volatiletech/sqlboiler/v4/queries/qm"

type QueryMods struct {
	Limit  *float64
	Offset *float64
}

func (qmt *QueryMods) GetQueryMods() []qm.QueryMod {
	var queryMods []qm.QueryMod

	if qmt.Limit != nil {
		queryMods = append(queryMods, qm.Limit(int(*qmt.Limit)))
	}

	if qmt.Offset != nil {
		queryMods = append(queryMods, qm.Offset(int(*qmt.Offset)))
	}

	return queryMods
}
