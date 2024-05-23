# reEgg-go
A server emulator for Auxbrain's Egg, Inc. mobile game, written in Go, for version 1.12.13.  
This project was created to encourage even more people to [join the effort to emulate all the servers](https://based.quest/reverse-engineering-a-mobile-app-protobuf-api/).  
He is also the one that did all the hard work in researching and writing the first emulator. This one here is for further education and for people who prefer Go over Python.

## Why is this Game specifically chosen?
This game, and version specifically, communicates only over unsecured HTTP and the game files contain the protobuf definitions.  
That means all we have to do is to run a DNS redirecting service on the phone to point from "www.auxbrain.com" to a local ip address.  
If you are using an adblocker like [AdAway](https://f-droid.org/en/packages/org.adaway/), or alternatively [personalDNSfilter](https://f-droid.org/en/packages/dnsfilter.android/) if you have an older device, it is trivial to get started.  
The other reasons see the blogpost.

## Setup
- Install the App
- Install the adblocker and configure the redirect
- Run the server
- On first start the app asks you to accept the privacy policy. Click to read it which should open your phone's browser. If you end up at a page saying "Active Users" you did the dns redirect setup correctly. If you don't, figure out why.
- Play the game. It will incrementally unlock customised stuff.

## OG Roadmap
- [x] First contact
  - [x] Offer a valid backup
  - [x] Respond with valid payload
  - [x] Unlock Pro Permit
- [x] Gift Calendar
- [ ] Private Server API
  - [ ] Break Piggy Bank week after filling
  - [ ] Rename a device ID to a friendly name via API
  - [ ] Self-service dashboard & GUI
- [ ] Periodicals
  - [ ] Contracts
    - [x] Your First Contract
    - [x] Contract Scheduler
    - [x] Basic Co-op
      - [x] Persistence
      - [ ] Change Displayname (abuse coop contract farm?)
      - [ ] Computer Simulations
  - [x] Events
    - [x] Proof of Concept
    - [x] Event Scheduler

## My Extensions
- [x] First contact
  - [x] Give Soul Eggs to unlock contract view early for motd.
- [ ] Private Server API
  - [x] Simple Leaderboard
  - [ ] Account self service interface
    - [ ] Figure out how we would make this secure
  - [ ] Admin interface
  - [ ] Server Federation
- [ ] Periodicals
  - [x] Random Gift from Server
  - [x] Good Morning message (on first_contact)

## DISCLAIMER
The version of game chosen is deliberate to not affect the current live service. The game developer has had a history of having to
deal with cheaters and I do not wish to furthen the problem by attempting to reverse engineer their efforts against cheaters.  
The project's scope is to educate people on how to reverse engineer APIs in order for digital preservation and archival of media.  
As API servers shut down, many apps are immediately locked out from being useable or get heavily hindered in capabilities to do anything productive.

## Notes and Docs
Run on port 80 without sudo. Don't do this in production!  
sudo sysctl net.ipv4.ip_unprivileged_port_start=0
