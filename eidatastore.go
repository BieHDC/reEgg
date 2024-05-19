package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	ei "biehdc.reegg/eggpb"
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

func (egg *eggstore) Shutdown() {
	log.Println("save all your stuff now")
}
