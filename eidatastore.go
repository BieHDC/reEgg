package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	ei "biehdc.reegg/eggpb"
	"biehdc.reegg/lockmap"
	"google.golang.org/protobuf/encoding/protojson"
)

func (egg *eggstore) backupToFile(backup *ei.Backup) {
	userid := *backup.UserId
	fp := filepath.Join(egg.workingpath, fmt.Sprintf("savegame_%s.json", userid))

	bytes, err := protojson.Marshal(backup)
	if err != nil {
		log.Printf("couldnt marshal backup for %s: %s", userid, err)
		return
	}
	err = os.WriteFile(fp, bytes, 500)
	if err != nil {
		log.Printf("couldnt save backup for %s: %s", userid, err)
		return
	}

	//log.Printf("backup for %s saved to file %s", userid, fp)
}

func (egg *eggstore) backupFromFile(userid string) *ei.Backup {
	fp := filepath.Join(egg.workingpath, fmt.Sprintf("savegame_%s.json", userid))

	fr, err := os.ReadFile(fp)
	if err != nil {
		log.Printf("couldnt load backup for %s: %s", userid, err)
		return nil
	}

	var backup ei.Backup
	err = protojson.Unmarshal(fr, &backup)
	if err != nil {
		log.Printf("couldnt unmarshal backup for %s: %s", userid, err)
		return nil
	}

	return &backup
}

func (egg *eggstore) SaveData() {
	egg.saveMembersToFile()
	egg.saveCoopgamesToFile()
	egg.saveCoopgiftsToFile()
}

func (egg *eggstore) Shutdown() {
	log.Println("shutting down, saving all your stuff now")
	egg.SaveData()
}

func (egg *eggstore) saveMembersToFile() {
	fp := filepath.Join(egg.workingpath, "members.json")

	membersmap, unlocker := egg.members.LockAndGetUnderlyingMapWithUnlocker()
	defer unlocker()

	bytes, err := json.Marshal(membersmap)
	if err != nil {
		log.Printf("couldnt marshal members: %s", err)
		return
	}
	err = os.WriteFile(fp, bytes, 500)
	if err != nil {
		log.Printf("couldnt save members to file: %s", err)
		return
	}

	//log.Printf("members saved to file: %s", fp)
}

func (egg *eggstore) loadMembersFromFile() *lockmap.LockMap[string, []usermemberinfo] {
	// empty return map in case we fail to load
	lm := lockmap.MakeLockMap[string, []usermemberinfo](nil)

	fp := filepath.Join(egg.workingpath, "members.json")

	fr, err := os.ReadFile(fp)
	if err != nil {
		log.Printf("couldnt load members from file: %s", err)
		return lm
	}

	membersmap := make(map[string][]usermemberinfo)
	err = json.Unmarshal(fr, &membersmap)
	if err != nil {
		log.Printf("couldnt unmarshal members: %s", err)
		return lm
	}

	return lockmap.MakeLockMap[string, []usermemberinfo](&membersmap)
}

func (egg *eggstore) saveCoopgamesToFile() {
	fp := filepath.Join(egg.workingpath, "coopgames.json")

	coopgamesmap, unlocker := egg.coopgames.LockAndGetUnderlyingMapWithUnlocker()
	defer unlocker()

	bytes, err := json.Marshal(coopgamesmap)
	if err != nil {
		log.Printf("couldnt marshal coopgames: %s", err)
		return
	}
	err = os.WriteFile(fp, bytes, 500)
	if err != nil {
		log.Printf("couldnt save coopgames to file: %s", err)
		return
	}

	//log.Printf("coopgames saved to file: %s", fp)
}

func (egg *eggstore) loadCoopgamesFromFile() *lockmap.LockMap[string, contractGame] {
	// empty return map in case we fail to load
	lm := lockmap.MakeLockMap[string, contractGame](nil)

	fp := filepath.Join(egg.workingpath, "coopgames.json")

	fr, err := os.ReadFile(fp)
	if err != nil {
		log.Printf("couldnt load coopgames from file: %s", err)
		return lm
	}

	coopgamesmap := make(map[string]contractGame)
	err = json.Unmarshal(fr, &coopgamesmap)
	if err != nil {
		log.Printf("couldnt unmarshal coopgames: %s", err)
		return lm
	}

	return lockmap.MakeLockMap[string, contractGame](&coopgamesmap)
}

func (egg *eggstore) saveCoopgiftsToFile() {
	fp := filepath.Join(egg.workingpath, "coopgifts.json")

	coopgiftsmap, unlocker := egg.coopgifts.LockAndGetUnderlyingMapWithUnlocker()
	defer unlocker()

	bytes, err := json.Marshal(coopgiftsmap)
	if err != nil {
		log.Printf("couldnt marshal coopgifts: %s", err)
		return
	}
	err = os.WriteFile(fp, bytes, 500)
	if err != nil {
		log.Printf("couldnt save coopgifts to file: %s", err)
		return
	}

	//log.Printf("coopgifts saved to file: %s", fp)
}

func (egg *eggstore) loadCoopgiftsFromFile() *lockmap.LockMap[string, []*ei.ContractCoopStatusResponse_CoopGift] {
	// empty return map in case we fail to load
	lm := lockmap.MakeLockMap[string, []*ei.ContractCoopStatusResponse_CoopGift](nil)

	fp := filepath.Join(egg.workingpath, "coopgifts.json")

	fr, err := os.ReadFile(fp)
	if err != nil {
		log.Printf("couldnt load coopgifts from file: %s", err)
		return lm
	}

	coopgamesmap := make(map[string][]*ei.ContractCoopStatusResponse_CoopGift)
	err = json.Unmarshal(fr, &coopgamesmap)
	if err != nil {
		log.Printf("couldnt unmarshal coopgifts: %s", err)
		return lm
	}

	return lockmap.MakeLockMap[string, []*ei.ContractCoopStatusResponse_CoopGift](&coopgamesmap)
}
