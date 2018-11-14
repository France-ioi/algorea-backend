package database

import t "github.com/France-ioi/AlgoreaBackend/app/types"

// ItemStringStore implements database operations on `items_strings`
type ItemStringStore struct {
	db *DB
}

type ItemString struct {
	ID         t.Int64  `db:"ID"`
	ItemID     t.Int64  `db:"idItem"`
	LanguageID t.Int64  `db:"idLanguage"`
	Title      t.String `db:"sTitle"`
	Version    int64    `db:"iVersion"` // when the db does not know the default, they will get the go type default
}

func (s *ItemStringStore) createRaw(tx Tx, entry *ItemString) (int64, error) {
	if !entry.ID.Set {
		entry.ID = *t.NewInt64(generateID())
	}
	err := tx.insert("items_strings", entry)
	return entry.ID.Value, err
}
