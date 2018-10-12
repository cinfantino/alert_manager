package handler

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/mayuresh82/alert_manager/internal/models"
	"sort"
	"sync"
	"time"
)

const SUPPRULE_UPDATE_INTERVAL = 10 * time.Minute

// suppressor manages suppression rules and alert suppressions
type suppressor struct {
	suppRules models.SuppRules
	db        models.Dbase

	sync.Mutex
}

// Global Suppressor Singleton
var suppr *suppressor
var suppOnce sync.Once

func GetSuppressor(db models.Dbase) *suppressor {
	suppOnce.Do(func() {
		suppr = &suppressor{db: db}
		ctx := context.Background()
		suppr.loadSuppRules(ctx)
		go func() {
			t := time.NewTicker(SUPPRULE_UPDATE_INTERVAL)
			for range t.C {
				suppr.loadSuppRules(ctx)
			}
		}()
	})
	return suppr
}

func (s *suppressor) loadSuppRules(ctx context.Context) {
	s.Lock()
	defer s.Unlock()
	glog.V(2).Infof("Updating suppression rules")
	tx := s.db.NewTx()
	var (
		rules models.SuppRules
		er    error
	)
	err := models.WithTx(ctx, tx, func(ctx context.Context, tx models.Txn) error {
		if rules, er = tx.SelectRules(models.QuerySelectActive + " LIMIT 50"); er != nil {
			return er
		}
		return nil
	})
	if err != nil {
		glog.Errorf("Unable to select rules from db: %v", err)
	}
	s.suppRules = rules

	// load persistent rules from config
	for _, rule := range Config.GetSuppressionRules() {
		for k, v := range rule.Matches {
			ents := models.Labels{k: v}
			r := models.NewSuppRule(ents, rule.Type, rule.Reason, "alert manager", rule.Duration)
			r.DontExpire = true
			s.suppRules = append(s.suppRules, r)
		}
	}
}

func (s *suppressor) SaveRule(ctx context.Context, tx models.Txn, rule models.SuppressionRule) error {
	id, err := tx.NewSuppRule(&rule)
	if err != nil {
		return err
	}
	rule.Id = id
	s.Lock()
	defer s.Unlock()
	s.suppRules = append(s.suppRules, rule)
	return nil
}

func (s *suppressor) Match(labels models.Labels, cond models.MatchCondition) (models.SuppressionRule, bool) {
	s.Lock()
	defer s.Unlock()
	var matches []models.SuppressionRule
	for i, rule := range s.suppRules {
		if rule.Match(labels, cond) {
			matches = append(matches, rule)
			if rule.TimeLeft() <= 0 {
				// rule has expired, remove from cache
				s.suppRules = append(s.suppRules[:i], s.suppRules[i+1:]...)
			}
		}
	}
	// if more than one rules match, return the most recent
	if len(matches) > 0 {
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].CreatedAt.After(matches[j].CreatedAt.Time)
		})
		return matches[0], true
	}
	return models.SuppressionRule{}, false
}

func (s *suppressor) SuppressAlert(
	ctx context.Context,
	tx models.Txn,
	alert *models.Alert,
	rule models.SuppressionRule,
) error {
	alert.Suppress(time.Duration(rule.Duration) * time.Second)
	if err := tx.UpdateAlert(alert); err != nil {
		return fmt.Errorf("Unable to update alert: %v", err)
	}
	if err := s.SaveRule(ctx, tx, rule); err != nil {
		return fmt.Errorf("Unable to save rule: %v", err)
	}
	return nil
}

func (s *suppressor) UnsuppressAlert(ctx context.Context, tx models.Txn, alert *models.Alert) error {
	existing, err := tx.GetAlert(models.QuerySelectById, alert.Id)
	if err != nil {
		return err
	}
	if existing.Status != models.Status_SUPPRESSED {
		return fmt.Errorf("Alert %d has cleared or expired, not unsuppressing", existing.Id)
	}
	alert.Unsuppress()
	if err := tx.UpdateAlert(alert); err != nil {
		glog.Errorf("Failed up update alert status: %v", err)
		return err
	}
	return nil
}
