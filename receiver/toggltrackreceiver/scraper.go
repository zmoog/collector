package toggltrackreceiver

import (
	toggl "github.com/jason0x43/go-toggl"
	"go.uber.org/zap"
)

func NewScraper(apiToken string, logger *zap.Logger) *accountScraper {
	session := toggl.OpenSession(apiToken)

	return &accountScraper{
		session: &session,
		logger:  logger,
	}
}

// type scraper struct {
// 	session        *toggl.Session
// 	logger         *zap.Logger
// 	lastScrapeTime time.Time
// }

// func (s *scraper) Scrape(referenceTime time.Time, lookback time.Duration) (toggl.Account, []toggl.TimeEntry, error) {
// 	var endDate = referenceTime
// 	var startDate = endDate.Add(-lookback)

// 	// Get the account information (we're interested
// 	// in the projects and workspaces).
// 	account, err := s.session.GetAccount()
// 	if err != nil {
// 		return toggl.Account{}, nil, err
// 	}
// 	s.logger.Info("Account", zap.Any("account", account))

// 	// Get the time entries started between startDate and endDate.
// 	entries, err := s.session.GetTimeEntries(startDate, endDate)
// 	if err != nil {
// 		return toggl.Account{}, nil, err
// 	}

// 	// We only want to send the entries
// 	// we haven't processed before.
// 	var newEntries []toggl.TimeEntry

// 	for _, entry := range entries {
// 		if entry.IsRunning() {
// 			// we only want to
// 			// consider completed
// 			// entries.
// 			continue
// 		}

// 		if entry.Stop.After(s.lastScrapeTime) {
// 			// add the entry to the new entries
// 			newEntries = append(newEntries, entry)
// 		}
// 	}

// 	s.lastScrapeTime = endDate

// 	return account, newEntries, nil
// }

type accountScraper struct {
	session *toggl.Session
	logger  *zap.Logger
}

func (s *accountScraper) Scrape() (toggl.Account, error) {
	account, err := s.session.GetAccount()
	if err != nil {
		return toggl.Account{}, err
	}
	return account, nil
}
