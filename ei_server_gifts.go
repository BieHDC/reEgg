package main

import (
	"log"
	"time"

	ei "biehdc.reegg/eggpb"
	"google.golang.org/protobuf/proto"
)

// get_periodicals: user_id:"aaa" piggy_full:true piggy_found_full:false seconds_full_realtime:96802.13004922867 seconds_full_gametime:13304.21092633903 soul_eggs:1290.0000000000002 current_client_version:26 debug:false
func (egg *eggstore) serverGifts(pr *ei.GetPeriodicalsRequest) []*ei.ServerGift {
	if pr.UserId == nil {
		return nil
	}
	pending, _ := egg.pendinggifts.LoadAndDelete(*pr.UserId)
	//delete(egg.pendinggifts, *pr.UserId)

	backup, _ := egg.users.Load(*pr.UserId)
	gift := serverGiftEarningsboost(backup)
	if gift != nil {
		//log.Printf("user %s is getting free money: %f", *pr.UserId, *gift.RewardAmount)
		pending = append(pending, gift)
	}

	return pending
}

// random money
func serverGiftEarningsboost(backup *ei.Backup) *ei.ServerGift {
	if backup == nil {
		return nil
	}
	if backup.Game == nil {
		return nil
	}
	if backup.Game.CurrentFarm == nil {
		return nil
	}
	currentfarmid := *backup.Game.CurrentFarm

	if backup.Farms == nil {
		return nil
	}
	if backup.Farms[currentfarmid] == nil {
		return nil
	}
	farm := backup.Farms[currentfarmid]
	if farm.CashEarned == nil {
		return nil
	}
	cash := *farm.CashEarned

	var (
		reward = ei.RewardType_CASH
		amount = cash / 5 // idk yet, maybe?
	)
	casheggs := ei.ServerGift{
		RewardType:   &reward,
		RewardAmount: &amount,
	}

	return &casheggs
}

// make 250 soul eggs for contracts
func serverGiftSouleggs(amount float64) *ei.ServerGift {
	var reward = ei.RewardType_SOUL_EGGS

	souleggs := ei.ServerGift{
		RewardType:   &reward,
		RewardAmount: &amount,
	}

	return &souleggs
}

func path_daily_gift_info(_ []byte) []byte {
	today := time.Now()

	gametoday := today.Sub(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))
	currentday := uint32(gametoday.Hours() / 24)

	tomorrow := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()) //midnight today
	tomorrow.AddDate(0, 0, 1)                                                                     //midnight tomorrow
	secondstonext := 86400 - today.Sub(tomorrow).Seconds()

	dgi := ei.DailyGiftInfo{}
	dgi.CurrentDay = &currentday
	dgi.SecondsToNextDay = &secondstonext

	// log.Printf("dailygiftinfo: %s", dgi.String())

	resp, err := proto.Marshal(&dgi)
	if err != nil {
		log.Printf("failed to marshal DailyGiftInfo: %s", err.Error())
		return nil
	}
	return resp
}
