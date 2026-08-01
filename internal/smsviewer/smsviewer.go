package smsviewer

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"bfr-webui-go/internal/config"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type SMSMessage struct {
	ID       int64  `json:"id"`
	Address  string `json:"address"`
	Body     string `json:"body"`
	Date     int64  `json:"date"`
	DateSent int64  `json:"date_sent"`
	Read     int    `json:"read"`
	Type     int    `json:"type"`
}

type SMSResponse struct {
	Messages []SMSMessage `json:"messages"`
	Total    int          `json:"total"`
	Limit    int          `json:"limit"`
	Offset   int          `json:"offset"`
	Search   string       `json:"search"`
	Error    string       `json:"error,omitempty"`
}

var candidateDBPaths = []string{
	"/data/data/com.android.providers.telephony/databases/telephony.db",
	"/data/user_de/0/com.android.providers.telephony/databases/telephony.db",
	"/data/data/com.android.providers.telephony/databases/mmssms.db",
	"/data/user_de/0/com.android.providers.telephony/databases/mmssms.db",
}

func getDBPath() (string, func(), error) {
	paths := candidateDBPaths
	if config.SMSDb != "" {
		paths = append([]string{config.SMSDb}, candidateDBPaths...)
	}
	var targetPath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			targetPath = p
			break
		}
	}

	if targetPath == "" {
		// Try using su to check files
		for _, p := range paths {
			out, err := exec.Command(config.SUBin, "-c", fmt.Sprintf("test -f %s && echo 1 || echo 0", p)).Output()
			if err == nil && strings.TrimSpace(string(out)) == "1" {
				targetPath = p
				break
			}
		}
	}

	if targetPath == "" {
		return "", func() {}, fmt.Errorf("telephony.db not found on device")
	}

	// Try reading directly
	if _, err := os.Stat(targetPath); err == nil {
		if file, err := os.Open(targetPath); err == nil {
			file.Close()
			return targetPath, func() {}, nil
		}
	}

	// If direct read fails due to root permission, copy to temporary location in /data/local/tmp or current dir via su
	tmpPath := "/data/local/tmp/telephony_bfr_copy.db"
	// M-8: use chmod 600 (owner-read-write only) for the temp DB copy.
	cmdStr := fmt.Sprintf("%s -c 'cp %s %s && chmod 600 %s'", config.SUBin, targetPath, tmpPath, tmpPath)
	if err := exec.Command("sh", "-c", cmdStr).Run(); err == nil {
		cleanup := func() {
			_ = os.Remove(tmpPath)
		}
		return tmpPath, cleanup, nil
	}

	return targetPath, func() {}, nil
}

func ReadSMSViaContentProvider(limit int, offset int, searchQuery string) (SMSResponse, error) {
	// Execute Android content query command via su
	cmdStr := "content query --uri content://sms --projection _id,address,body,date,read,type"
	out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
	if err != nil {
		return SMSResponse{Messages: []SMSMessage{}, Limit: limit, Offset: offset, Search: searchQuery, Error: fmt.Sprintf("content query failed: %v", err)}, nil
	}

	lines := strings.Split(string(out), "\n")
	allMessages := make([]SMSMessage, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Row:") {
			continue
		}

		var msg SMSMessage
		// Parse Row format: Row: 0 _id=1, address=+628123, body=Hello, date=1710000000000, read=1, type=1
		parts := strings.Split(line, ", ")
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])

			// If key starts with Row: X _id
			if strings.Contains(key, "_id") {
				var id int64
				fmt.Sscanf(val, "%d", &id)
				msg.ID = id
			} else if key == "address" {
				msg.Address = val
			} else if key == "body" {
				msg.Body = val
			} else if key == "date" {
				var d int64
				fmt.Sscanf(val, "%d", &d)
				if d > 9999999999 {
					d = d / 1000
				}
				msg.Date = d
			} else if key == "read" {
				var r int
				fmt.Sscanf(val, "%d", &r)
				msg.Read = r
			} else if key == "type" {
				var t int
				fmt.Sscanf(val, "%d", &t)
				msg.Type = t
			}
		}

		if searchQuery != "" {
			qLower := strings.ToLower(searchQuery)
			if !strings.Contains(strings.ToLower(msg.Address), qLower) && !strings.Contains(strings.ToLower(msg.Body), qLower) {
				continue
			}
		}

		if msg.Address != "" || msg.Body != "" {
			allMessages = append(allMessages, msg)
		}
	}

	total := len(allMessages)
	// Apply limit and offset pagination
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	sliced := allMessages[start:end]

	return SMSResponse{
		Messages: sliced,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		Search:   searchQuery,
	}, nil
}

func ReadSMSInbox(limit int, offset int, searchQuery string) (SMSResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Primary method: ContentProvider query (works 100% reliably on Android 10-16 ROMs without file locking)
	cpResp, cpErr := ReadSMSViaContentProvider(limit, offset, searchQuery)
	if cpErr == nil && len(cpResp.Messages) > 0 {
		return cpResp, nil
	}

	// Fallback to direct SQLite DB query
	dbPath, cleanup, err := getDBPath()
	if err != nil {
		if len(cpResp.Messages) > 0 {
			return cpResp, nil
		}
		return SMSResponse{Messages: []SMSMessage{}, Limit: limit, Offset: offset, Search: searchQuery, Error: err.Error()}, nil
	}
	defer cleanup()

	dsn := fmt.Sprintf("file:%s?mode=ro", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return SMSResponse{Messages: []SMSMessage{}, Limit: limit, Offset: offset, Search: searchQuery, Error: fmt.Sprintf("failed opening db: %v", err)}, nil
	}
	defer db.Close()

	var whereClause string
	var args []interface{}

	searchQuery = strings.TrimSpace(searchQuery)
	if searchQuery != "" {
		whereClause = " WHERE address LIKE ? OR body LIKE ?"
		pattern := "%" + searchQuery + "%"
		args = append(args, pattern, pattern)
	}

	// Count total query
	countQuery := "SELECT COUNT(*) FROM sms" + whereClause
	var total int
	err = db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		// Table might be named 'sms' or 'messages' in some ROMs
		// fallback to check sms table
		total = 0
	}

	// Fetch messages
	selectQuery := fmt.Sprintf("SELECT _id, address, body, date, date_sent, read, type FROM sms%s ORDER BY date DESC LIMIT ? OFFSET ?", whereClause)
	queryArgs := append(args, limit, offset)

	rows, err := db.Query(selectQuery, queryArgs...)
	if err != nil {
		return SMSResponse{
			Messages: []SMSMessage{},
			Total:    total,
			Limit:    limit,
			Offset:   offset,
			Search:   searchQuery,
			Error:    fmt.Sprintf("query error: %v", err),
		}, nil
	}
	defer rows.Close()

	messages := make([]SMSMessage, 0)
	for rows.Next() {
		var msg SMSMessage
		var addr, body sql.NullString
		var date, dateSent sql.NullInt64
		var read, msgType sql.NullInt64

		if err := rows.Scan(&msg.ID, &addr, &body, &date, &dateSent, &read, &msgType); err != nil {
			continue
		}

		msg.Address = addr.String
		msg.Body = body.String
		msg.Date = date.Int64
		msg.DateSent = dateSent.Int64
		msg.Read = int(read.Int64)
		msg.Type = int(msgType.Int64)

		if msg.Date > 9999999999 {
			msg.Date = msg.Date / 1000
		}
		if msg.DateSent > 9999999999 {
			msg.DateSent = msg.DateSent / 1000
		}

		messages = append(messages, msg)
	}

	return SMSResponse{
		Messages: messages,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		Search:   searchQuery,
	}, nil
}
