package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChinX/helloworld/rest/common/config"
	"github.com/ChinX/helloworld/rest/common/servicecenter/v3"
)

var (
	HeartbeatInterval = 30 * time.Second
	serviceID         string
	instanceID        string
	instanceID1        string
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go run(ctx)
	fmt.Println("awaiting signal")
	sig := <-sigChan
	cancel()
	fmt.Println()
	fmt.Println("close instance by:", sig)
	destroy()
	fmt.Println("exiting")
}

func destroy() {
	cli := v3.NewClient(config.Registry.Address, config.Tenant.Domain)
	err := cli.UnRegisterInstance(serviceID, instanceID)
	if err != nil{
		log.Println(err)
		return
	}
	time.Sleep(time.Second * 3)
	err = cli.UnRegisterService(serviceID)
	if err != nil{
		log.Println(err)
	}
	return
}

func run(ctx context.Context) {
	// 加载配置文件
	err := config.LoadConfig("./conf/microservice.yaml")
	if err != nil {
		log.Fatalf("load config file faild: %s", err)
	}
	// 注册微服务与实例，启动心跳
	go registerAndHeartbeat(ctx)

	// 启动 http 监听
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	})
	err = http.ListenAndServe(config.Instance.ListenAddress, nil)

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	})
	err = http.ListenAndServe(config.Instance1.ListenAddress, nil)

	log.Println(err)
}

func registerAndHeartbeat(ctx context.Context) {
	// 微服务未注册则注册其信息
	cli := v3.NewClient(config.Registry.Address, config.Tenant.Domain)
	svcID, _ := cli.GetServiceID(config.Service)
	if svcID == "" {
		var err error
		svcID, err = cli.RegisterService(config.Service)
		if err != nil {
			log.Fatalln(err)
		}
	}

	serviceID = svcID

	// 注册微服务实例
	insID, err := cli.RegisterInstance(svcID, config.Instance)
	if err != nil {
		log.Fatalln(err)
	}

	instanceID = insID


	go func() {
		// 启动定时器：间隔30s
		tk := time.NewTicker( 30 * time.Second)
		for {
			select {
			case <-tk.C:
				// 定时发送心跳
				err := cli.Heartbeat(svcID, instanceID)
				if err != nil {
					log.Println(err)
					tk.Stop()
					return
				}
				log.Println("send heartbeat success",instanceID)
				// 监听程序退出
			case <-ctx.Done():
				tk.Stop()
				log.Println("service is done")
				return
			}
		}
	}()


	log.Println("再注册个注册微服务实例")
	// 再注册个注册微服务实例
	insID, err = cli.RegisterInstance(svcID, config.Instance1)
	if err != nil {
		log.Fatalln(err)
	}

	instanceID1 = insID

	// 启动定时器：间隔30s
	tk1 := time.NewTicker( 30 * time.Second)
	for {
		select {
		case <-tk1.C:
			// 定时发送心跳
			err := cli.Heartbeat(svcID, instanceID1)
			if err != nil {
				log.Println(err)
				tk1.Stop()
				return
			}
			log.Println("send heartbeat success1.",instanceID1)
			// 监听程序退出
		case <-ctx.Done():
			tk1.Stop()
			log.Println("service is done1")
			return
		}
	}

}
