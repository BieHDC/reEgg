package main

import (
	"log"
	"time"

	ei "biehdc.reegg/eggpb"
)

// raw ingameEvent data
type ingameEvent struct {
	type_      string
	internal   string
	subtitle   string // can be controlled by us
	multiplier float64
}

var rawevents = []ingameEvent{
	// when day == 1 -> use 0 and 1, double events
	{"vehicle-sale", "VEHICLE SALE", "75% off all vehicles", 0.25},
	{"drone-boost", "GENEROUS DRONES", "5x drone rewards", 5.0},
	// 2 and onwards map directly to day in month
	{"epic-research-sale", "EPIC RESEARCH SALE", "30% off Epic Research Upgrades", 0.7},
	{"research-sale", "RESEARCH SALE", "70% off Research Upgrades", 0.3},
	{"hab-sale", "HAB SALE", "70% OFF HEN HOUSES!", 0.3},
	{"boost-sale", "BOOST SALE", "30% OFF BOOSTS!", 0.7},
	{"earnings-boost", "CASH BOOST", "3x EARNINGS!", 3.0},
	{"gift-boost", "GENEROUS GIFTS", "4x GIFTS!", 4.0},
	{"boost-duration", "BOOST TIME+", "DOUBLE BOOST TIME", 2.0},
	{"hab-sale", "HAB SALE", "70% OFF HEN HOUSES!", 0.3},
	{"vehicle-sale", "VEHICLE SALE", "75% off all vehicles", 0.25},
	{"drone-boost", "GENEROUS DRONES", "5x drone rewards", 5.0},
	{"research-sale", "RESEARCH SALE", "70% off Research Upgrades", 0.3},
	{"boost-sale", "BOOST SALE", "30% OFF BOOSTS!", 0.7},
	{"earnings-boost", "CASH BOOST", "3x EARNINGS!", 3.0},
	{"hab-sale", "HAB SALE", "70% OFF HEN HOUSES!", 0.3},
	{"boost-duration", "BOOST TIME+", "DOUBLE BOOST TIME", 2.0},
	{"gift-boost", "GENEROUS GIFTS", "4x GIFTS!", 4.0},
	{"epic-research-sale", "EPIC RESEARCH SALE", "30% off Epic Research Upgrades", 0.7},
	{"vehicle-sale", "VEHICLE SALE", "75% off all vehicles", 0.25},
	{"drone-boost", "GENEROUS DRONES", "5x drone rewards", 5.0},
	{"boost-duration", "BOOST TIME+", "DOUBLE BOOST TIME", 2.0},
	{"research-sale", "RESEARCH SALE", "70% off Research Upgrades", 0.3},
	{"earnings-boost", "CASH BOOST", "3x EARNINGS!", 3.0},
	{"hab-sale", "HAB SALE", "70% OFF HEN HOUSES!", 0.3},
	{"gift-boost", "GENEROUS GIFTS", "4x GIFTS!", 4.0},
	{"vehicle-sale", "VEHICLE SALE", "75% off all vehicles", 0.25},
	{"drone-boost", "GENEROUS DRONES", "5x drone rewards", 5.0},
	{"earnings-boost", "CASH BOOST", "3x EARNINGS!", 3.0},
	{"epic-research-sale", "EPIC RESEARCH SALE", "30% off Epic Research Upgrades", 0.7},
	{"boost-duration", "BOOST TIME+", "DOUBLE BOOST TIME", 2.0},
	{"hab-sale", "HAB SALE", "70% OFF HEN HOUSES!", 0.3},
}
var events []*ei.EggIncEvent

// special events
var double_prestige_event_raw = ingameEvent{"prestige-boost", "PRESTIGE BOOST", "DOUBLE PRESTIGE!", 2.0}
var triple_prestige_event_raw = ingameEvent{"prestige-boost", "PRESTIGE BOOST", "TRIPLE PRESTIGE!", 3.0}
var double_prestige_event *ei.EggIncEvent
var triple_prestige_event *ei.EggIncEvent

type userEvent struct {
	message string
	timeout float64
}

var motd_raw = ingameEvent{"earnings-boost", "HELLO logcat", "TO BE FILLED BY current_events", 1.0}
var motd_event *ei.EggIncEvent
var usernotification = ingameEvent{"epic-research-sale", "NOTIFICATION", "ALSO TO BE FILLED BY current_events", 1.0}

// convert the raw stuff into protobuf things
func init() {
	for _, evt := range rawevents {
		events = append(events, &ei.EggIncEvent{
			Identifier: &evt.internal,
			//SecondsRemaining filled in by current_events
			Type:       &evt.type_,
			Multiplier: &evt.multiplier,
			Subtitle:   &evt.subtitle,
		})
	}
	double_prestige_event = &ei.EggIncEvent{
		Identifier: &double_prestige_event_raw.internal,
		//SecondsRemaining filled by current_events
		Type:       &double_prestige_event_raw.type_,
		Multiplier: &double_prestige_event_raw.multiplier,
		Subtitle:   &double_prestige_event_raw.subtitle,
	}
	triple_prestige_event = &ei.EggIncEvent{
		Identifier: &triple_prestige_event_raw.internal,
		//SecondsRemaining filled by current_events
		Type:       &triple_prestige_event_raw.type_,
		Multiplier: &triple_prestige_event_raw.multiplier,
		Subtitle:   &triple_prestige_event_raw.subtitle,
	}
	motd_event = &ei.EggIncEvent{
		Identifier: &motd_raw.internal,
		//SecondsRemaining filled by current_events
		Type:       &motd_raw.type_,
		Multiplier: &motd_raw.multiplier,
		Subtitle:   &motd_raw.subtitle,
	}
}

// 1st day in the month has special handling
func grabTodaysEvents(day int) []*ei.EggIncEvent {
	if day < 1 || day >= len(events) {
		log.Panicf("not supposed to happen. day was %d and len of events is %d", day, len(events))
	}
	if day == 1 {
		return []*ei.EggIncEvent{events[0], events[1]}
	}
	return []*ei.EggIncEvent{events[day]}
}

func (egg *eggstore) addEvent(userid, message string, timeout float64) {
	tmp, _ := egg.notificationevent.Load(userid)
	tmp = append(tmp,
		userEvent{
			message: message,
			timeout: timeout,
		})
	egg.notificationevent.Store(userid, tmp)
}

func (egg *eggstore) currentEvents(pr *ei.GetPeriodicalsRequest) *ei.EggIncCurrentEvents {
	today := time.Now()
	tomorrow := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()) //midnight today
	tomorrow.AddDate(0, 0, 1)                                                                     //midnight tomorrow
	secondslefttoday := 86400 - today.Sub(tomorrow).Seconds()

	day := today.Day()
	todaysevents := grabTodaysEvents(day)

	// it's a sunday, we have a fixed prestige event
	if today.Weekday() == time.Sunday {
		// ratio made up by me
		if day%3 == 0 {
			todaysevents = append(todaysevents, triple_prestige_event)
		} else {
			todaysevents = append(todaysevents, double_prestige_event)
		}
	}

	// set the correct timers
	for i := range todaysevents {
		todaysevents[i].SecondsRemaining = &secondslefttoday
	}

	// we do this here separatly so i can put a shorter timer, so it does not show up all the time
	var timeout = 20.0
	motd_event.Subtitle = &egg.motd
	motd_event.SecondsRemaining = &timeout
	todaysevents = append(todaysevents, motd_event)

	if pr.UserId != nil {
		notifications, _ := egg.notificationevent.LoadAndDelete(*pr.UserId)
		for _, notif := range notifications {
			newevent := ei.EggIncEvent{
				Identifier: &usernotification.internal,
				Type:       &usernotification.type_,
				Multiplier: &usernotification.multiplier,
				//
				Subtitle:         &notif.message,
				SecondsRemaining: &notif.timeout,
			}
			todaysevents = append(todaysevents, &newevent)
		}
		//delete(egg.notificationevent, *pr.UserId) //we have load and delete
	}

	eventstructure := ei.EggIncCurrentEvents{Events: todaysevents}
	return &eventstructure
}
