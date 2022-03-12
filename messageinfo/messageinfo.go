package messageinfo

type MessageInfo struct {
	Uid        uint32
	Sender     string
	Senders    []string
	Recipient  string // Not a recomended filter
	Recipients []string
	Cc         []string
	Bcc        []string
	Date       int64
	Day        int
	Weekday    string
	Month      int
	Monthname  string
	Year       int
	Subject    string
	Flags      []string
}
