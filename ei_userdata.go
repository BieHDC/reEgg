package main

import (
	"log"

	ei "biehdc.reegg/eggpb"
	syncmap "biehdc.reegg/genericsyncmap"
	"biehdc.reegg/lockmap"
	"google.golang.org/protobuf/proto"
)

type eggstore struct {
	motd string
	//
	users             syncmap.Map[string, *ei.Backup]
	notificationevent syncmap.Map[string, []userEvent]
	pendinggifts      syncmap.Map[string, []*ei.ServerGift]
	//
	workingpath string
	//
	members   *lockmap.LockMap[string, []usermemberinfo]
	coopgames *lockmap.LockMap[string, contractGame]
}

func newEggstore(motd, workingpath string) eggstore {
	generateContracts(workingpath)
	return eggstore{
		motd:        motd,
		workingpath: workingpath,
		//
		members:   lockmap.MakeLockMap[string, []usermemberinfo](),
		coopgames: lockmap.MakeLockMap[string, contractGame](),
	}
}

func (egg *eggstore) path_first_contact(decoded []byte) []byte {
	fcreq := ei.EggIncFirstContactRequest{}
	err := proto.Unmarshal(decoded, &fcreq)
	if err != nil {
		log.Printf("cant unmarshal EggIncFirstContactRequest: %s", err)
		return nil
	}
	if fcreq.UserId == nil {
		return nil
	}

	log.Printf("first contact: %s", fcreq.String())
	egg.addEvent(*fcreq.UserId, "GOOD MORNING MATE!!", 60.0)

	// check for existing backup
	var backup *ei.Backup
	var exists bool
	backup, exists = egg.users.Load(*fcreq.UserId)
	if exists {
		//log.Println("user has a hot savegame, returning it")
	} else {
		backup = egg.backupFromFile(*fcreq.UserId)
		if backup == nil {
			log.Println("no savegave for user")
		} else {
			//log.Println("loaded cold savegave for user")
			egg.users.Store(*fcreq.UserId, backup)
		}
	}

	fcresp := ei.EggIncFirstContactResponse{
		Backup: backup,
	}
	resp, err := proto.Marshal(&fcresp)
	if err != nil {
		log.Printf("failed to marshal EggIncFirstContactResponse: %s", err.Error())
		return nil
	}
	return resp
}

func (egg *eggstore) path_save_backup(decoded []byte) []byte {
	backup := ei.Backup{}
	err := proto.Unmarshal(decoded, &backup)
	if err != nil {
		log.Printf("cant unmarshal Backup: %s", err)
	}
	//js, _ := json.MarshalIndent(backup, "", "  ")
	//log.Printf("Backup:\n%s", js)
	if backup.UserId == nil {
		log.Printf("someone without userid requested save_backup")
		return nil
	}

	log.Printf("Backup for %s", *backup.UserId)
	if backup.Game != nil { // can be optional for some reason, is likely not
		// we do a little backup tuning
		// enable pro features
		notyetupgraded := false
		if backup.Game.PermitLevel == nil {
			permitlevel := uint32(1)
			backup.Game.PermitLevel = &permitlevel
			notyetupgraded = true
		}
		if *backup.Game.PermitLevel == 0 {
			*backup.Game.PermitLevel = 1
			notyetupgraded = true
		}
		if notyetupgraded {
			backup.ForceBackup = &notyetupgraded
			backup.ForceOfferBackup = &notyetupgraded
			egg.addEvent(*backup.UserId, "SAVE GAME UPGRADED!!\nRESTART THE APP, WAIT FOR THE LOAD BACKUP POPUP AND LOAD IT!", 999999.0)
			log.Println("Backup was spiced up!")
		} else {
			// we only give out the eggs to unlock contracts after we have loaded the spiced up backup
			// this should lead to a more conistent entry experience and less race conditions
			// sometimes still does double gifting, maybe needs a timeout to recheck
			// give enough eggs to enable contracts for server motd
			totaleggs := 0.0
			if backup.Game.SoulEggsD != nil {
				totaleggs += *backup.Game.SoulEggsD
			}
			if backup.Game.UnclaimedSoulEggsD != nil {
				totaleggs += *backup.Game.UnclaimedSoulEggsD
			}
			if totaleggs < 250.0 {
				log.Printf("%s has only %.0f eggs", *backup.UserId, totaleggs)
				// fill up the rest with eggs so we unlock contracts
				current, _ := egg.pendinggifts.Load(*backup.UserId)
				have := false
				for _, gifts := range current {
					if *gifts.RewardType == ei.RewardType_SOUL_EGGS {
						// we already have one pending
						log.Printf("%s already has a pending soul package for %.0f eggs", *backup.UserId, *gifts.RewardAmount)
						have = true
						break
					}
				}
				if !have {
					// very cheap way to avoid dupes
					current = append(current, serverGiftSouleggs(250.0-totaleggs))
					egg.pendinggifts.Store(*backup.UserId, current)
					log.Printf("Player was gifted %.0f eggs", 250.0-totaleggs)
				}
			}
		}
	}
	egg.users.Store(*backup.UserId, &backup)
	egg.backupToFile(&backup)

	return nil
}

// menu -> settings -> more
func (egg *eggstore) path_user_data_info(decoded []byte) []byte {
	udireq := ei.UserDataInfoRequest{}
	err := proto.Unmarshal(decoded, &udireq)
	if err != nil {
		log.Printf("cant unmarshal Backup: %s", err)
	}
	if udireq.UserId == nil || udireq.DeviceId == nil || udireq.BackupChecksum == nil {
		log.Println("bad user data info request")
		return nil
	}

	backup, _ := egg.users.Load(*udireq.UserId)
	if backup == nil {
		log.Printf("no backup for %s yet", *udireq.UserId)
		return nil
	}

	udiresp := ei.UserDataInfoResponse{
		BackupChecksum:  backup.Checksum,
		BackupTotalCash: backup.Game.LifetimeCashEarned,
		CoopMemberships: egg.getCoopMemberships(*udireq.UserId),
	}

	resp, err := proto.Marshal(&udiresp)
	if err != nil {
		log.Printf("failed to marshal EggIncFirstContactResponse: %s", err.Error())
		return nil
	}
	return resp
}
