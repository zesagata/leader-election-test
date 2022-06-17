package main

import (
	"context"
	"flag"
	"fmt"
	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	LeaderKey   = "leader-election-key"
	WaitTime    = 1 * time.Second
	WorkTime    = 5 * time.Second
	StandByTime = 5 * time.Second
)

func main() {

	name := flag.String("name", "", "Service name")
	flag.Parse()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
	})
	if err != nil {
		panic(err)
	}

	session, err := concurrency.NewSession(cli, concurrency.WithTTL(10))
	if err != nil {
		panic(err)
	}
	election := concurrency.NewElection(session, LeaderKey)

	for {
		isLeader := DoOrWaitElection(election, *name)
		if isLeader {
			fmt.Println("Im a leader ", *name)
			DoSomeWork(election)
		} else {
			fmt.Println("Im a follower, standby...")
			//time.Sleep(StandByTime)
		}
	}
}

func DoSomeWork(election *concurrency.Election) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)
	signal.Notify(sigs, syscall.SIGINT)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		for s := range sigs {
			fmt.Println("Got exit signal", s)
			err := election.Resign(context.Background())
			if err != nil {
				log.Fatalln("Failed to init election")
			}
			os.Exit(0)
		}

	}()
	for {
		leaderKey, _ := election.Leader(context.Background())
		fmt.Println(leaderKey)
		fmt.Println("Do some work , ", election.Key())
		time.Sleep(WorkTime)
	}
}

func DoOrWaitElection(election *concurrency.Election, name string) bool {
	ctx, _ := context.WithTimeout(context.Background(), WaitTime)
	err := election.Campaign(ctx, name)

	if err != nil && err == context.DeadlineExceeded {
		return false
	}
	return true
}
