package main

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/pikans/mealplan"
)

type ReminderGroup struct {
	Duties          []string
	ImportantDuties []string
	TodayText       string
}

var DutyGroups = map[string]ReminderGroup{
	"cook":  ReminderGroup{[]string{"Big Cook", "Little Cook", "Tiny Cook"}, []string{"Big Cook", "Little Cook"}, "today"},
	"clean": ReminderGroup{[]string{"Cleaner 1", "Cleaner 2", "Cleaner 3"}, []string{"Cleaner 1", "Cleaner 2"}, "tonight"},
}

func dayDeltaString(dayDelta int, todayText string) string {
	switch {
	case dayDelta == 0:
		return todayText
	case dayDelta == 1:
		return "tomorrow"
	case dayDelta == -1:
		return "yesterday"
	case dayDelta > 1:
		return fmt.Sprintf("in %d days", dayDelta)
	case dayDelta < -1:
		return fmt.Sprintf("%d days ago", -dayDelta)
	default:
		panic("impossible")
	}
}

const mailserver = "outgoing.mit.edu:smtp"
const from = "yfnkm@mit.edu"

func sendReminder(to []string, task string, mightBeCanceled bool) {
	msg :=
		`From: "pika kitchen manager" <%s>
To: %s
Subject: Reminder: you are signed up to %s

http://mealplan.pikans.org/

	`
	if mightBeCanceled {
		msg += "NOTE: not all shifts are filled, so dinner may be canceled"
	}
	body := fmt.Sprintf(msg, from, strings.Join(to, ", "), task)
	to = append(to, from) // bcc yfnkm
	err := smtp.SendMail(mailserver, nil, from, to, []byte(body))
	if err != nil {
		log.Printf("%v", err)
	}
}



// Returns whether any shifts are missing
func mightBeCanceled(data *Data, day string, group ReminderGroup) bool {
	dayAssignments, ok := data.Assignments[day]
	if !ok {
		return true
	}
	for _, duty := range group.ImportantDuties {
		assignee, ok := dayAssignments[duty]
	 	if !ok || assignee == "" {
			return true
		}
	}
	return false
}

func toEmail(username string) string {
	if strings.Contains(username, "@") {
		return username
	} else {
		return username + "@mit.edu"
	}
}

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("wrong number of args: run %v <datapath> <duty> <daysOut>", os.Args[0])
	}
	var ok bool
	var err error
	var group ReminderGroup
	var dayDelta int

	task := os.Args[2]
	if group, ok = DutyGroups[task]; !ok {
		log.Fatalf("no task '%s'", task)
	}

	if dayDelta, err = strconv.Atoi(os.Args[3]); err != nil {
		log.Fatalf("invalid day delta '%s': %v", os.Args[2], err)
	}
	
	data, err := ReadData(os.Args[1])
	if err != nil {
		log.Fatalf("couldn't read data from '%s': %v", os.Args[1], err)
	}

	to := []string{}

	day := time.Now().AddDate(0, 0, dayDelta).Format(DateFormat)
	dayAssignments, ok := data.Assignments[day]
	if ok {
		for _, duty := range group.Duties {
			assignee, ok := dayAssignments[duty]
			if ok && string(assignee) != "" {
				to = append(to, toEmail(string(assignee)))
			}
		}
	}
	taskText := fmt.Sprintf("%s %s", task, dayDeltaString(dayDelta, group.TodayText))
	sendReminder(to, taskText, mightBeCanceled(data, day, group))
}
