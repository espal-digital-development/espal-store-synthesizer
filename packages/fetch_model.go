package packages

const fetchModel = `func (#STRUCT_VAR_NAME *#STRUCT_NAME) fetch(query string, withCreators bool, params ...interface{}) (result []*#ENTITY_STRUCT_NAME, ok bool, err error) {
	rows, err := #STRUCT_VAR_NAME.selecterDatabase.Query(query, params...)
	if err == sql.ErrNoRows {
		err = nil
		return
	}
	if err != nil {
		err = errors.Trace(err)
		return
	}
	defer func(dbRows database.Rows) {
		closeErr := dbRows.Close()
		if err != nil && closeErr != nil {
			err = errors.Wrap(err, closeErr)
		} else if closeErr != nil {
			err = errors.Trace(closeErr)
		}
	}(rows)
	result = make([]*#ENTITY_STRUCT_NAME, 0)
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, false, errors.Trace(err)
		}
		#ENTITY_STRUCT_VAR_NAME := new#ENTITY_STRUCT_NAME()
		fields := []interface{}{#ENTITY_FIELDS}
		if withCreators {
			fields = append(fields, #ENTITY_CREATOR_FIELDS)
		}
		if err := rows.Scan(fields...); err != nil {
			return nil, false, errors.Trace(err)
		}
		result = append(result, #ENTITY_STRUCT_VAR_NAME)
	}
	ok = len(result) > 0
	return
}
`
