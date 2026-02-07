package main

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
)

func ensureCollection(app core.App, typ, name string, configure func(c *core.Collection) error) (*core.Collection, error) {
	existing, err := app.FindCollectionByNameOrId(name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var collection *core.Collection
	if existing == nil {
		collection = core.NewCollection(typ, name)
	} else {
		collection = existing
	}

	if err := configure(collection); err != nil {
		return nil, err
	}

	if err := app.Save(collection); err != nil {
		return nil, err
	}

	return collection, nil
}

func addFieldIfMissing(collection *core.Collection, field core.Field) {
	if collection.Fields.GetByName(field.GetName()) != nil {
		return
	}
	collection.Fields.Add(field)
}

func addIndexIfMissing(collection *core.Collection, index string) {
	for _, existing := range collection.Indexes {
		if existing == index {
			return
		}
	}
	collection.Indexes = append(collection.Indexes, index)
}

func setRuleIfNil(ptr **string, rule string) {
	if *ptr != nil {
		return
	}
	value := rule
	*ptr = &value
}
