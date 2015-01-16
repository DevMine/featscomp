// Copyright 2014-2015 The DevMine authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package features

import (
	"database/sql"
	"errors"

	"github.com/DevMine/featscomp/util"
)

type averageForks struct {
	name string
	db   *sql.DB
}

var _ Feature = (*averageForks)(nil)

// NewAverageForks creates a new average forks feature.
func NewAverageForks(name string, db *sql.DB) (*averageForks, error) {
	if name == "" {
		return nil, errors.New("averageForks: name cannot be empty")
	}
	if db == nil {
		return nil, errors.New("averageForks: db cannot be nil")
	}

	return &averageForks{name: name, db: db}, nil
}

// Score computes the scores of the average forks feature.
func (a averageForks) Score() error {
	if err := deleteOldScores(a.db, a.name); err != nil {
		return err
	}

	rows, err := a.db.Query(
		`SELECT SUM(ghr.forks_count), COUNT(r.id), u.id
         FROM users AS u
         LEFT JOIN users_repositories AS ur ON u.id = ur.user_id
         LEFT JOIN repositories AS r ON ur.repository_id = r.id
         LEFT JOIN gh_repositories AS ghr ON r.id = ghr.repository_id
         WHERE NOT ghr.fork
         GROUP BY u.id
         ORDER BY u.id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	featID, err := util.FindFeatureID(a.db, a.name)
	if err != nil {
		return err
	}

	var avgs []float64
	var userIDs []int64
	var maxVal float64
	for rows.Next() {
		var userID int64
		var forksCount, reposCount *int64
		if err := rows.Scan(&forksCount, &reposCount, &userID); err != nil {
			return err
		}

		var avg float64
		if (reposCount != nil) && (forksCount != nil) && *reposCount != 0 {
			avg = float64(*forksCount) / float64(*reposCount)
		}
		if avg > maxVal {
			maxVal = avg
		}
		avgs = append(avgs, avg)
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	coeff := 1.0 / maxVal
	for i, av := range avgs {
		if _, err = a.db.Exec(
			`INSERT INTO scores(user_id, feature_id, score)
             VALUES($1, $2, $3)`,
			userIDs[i], featID, av*coeff); err != nil {
			return err
		}
	}

	return nil
}

// Name returns the name of the feature.
func (a averageForks) Name() string {
	return a.name
}
