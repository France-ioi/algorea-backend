package database

import (
	t "github.com/France-ioi/AlgoreaBackend/app/types"
)

// ItemStore implements database operations on items
type ItemStore struct {
	db *DB
}

type Item struct {
	ID            t.Int64  `db:"ID"`
	Type          t.String `db:"sType"`
	TeamsEditable bool     `db:"bTeamsEditable"` // when the db does not know the default, they will get the go type default
	NoScore       bool     `db:"bNoScore"`       // when the db does not know the default, they will get the go type default
	Version       int64    `db:"iVersion"`       // when the db does not know the default, they will get the go type default
}

// Create insert an Item row in the database and associted values in related tables if needed
func (s *ItemStore) Create(item *Item, languageID t.Int64, title t.String, parentID t.Int64, order t.Int64) error {

	groupItemStore := &GroupItemStore{s.db}
	itemItemStore := &ItemItemStore{s.db}
	itemStringStore := &ItemStringStore{s.db}

	return s.db.inTransaction(func(tx Tx) error {
		itemID, err := s.createRaw(tx, item)
		itemIDt := *t.NewInt64(itemID)
		if err != nil {
			return err
		}
		if _, err = groupItemStore.createRaw(tx, &GroupItem{ItemID: itemIDt}); err != nil {
			return err
		}
		if _, err = itemStringStore.createRaw(tx, &ItemString{ItemID: itemIDt, LanguageID: languageID, Title: title}); err != nil {
			return err
		}
		if _, err = itemItemStore.createRaw(tx, &ItemItem{ChildItemID: itemIDt, Order: order}); err != nil {
			return err
		}
		return nil
	})
}

// createRaw insert a row in the transaction and returns the
func (s *ItemStore) createRaw(tx Tx, entry *Item) (int64, error) {
	if !entry.ID.Set {
		entry.ID = *t.NewInt64(generateID())
	}
	err := tx.insert("items", entry)
	return entry.ID.Value, err
}
