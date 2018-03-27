package main

import (
	"log"
	"net/http"
	"os"
	"time"

	serviceLog "github.com/go-kit/kit/log"
	transport "github.com/go-kit/kit/transport/http"
)

func main() {
	// Scanner bits
	requiredEnv := []string{
		"XCAT_API_SERVER",
		"XCAT_TOKEN",
		"MONGO_HOST",
		"MONGO_USER",
		"MONGO_PASSWD",
		"MONGO_DB",
		"MONGO_COLLECTION",
	}

	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			log.Fatalf("unable to find %s, exit", env)
		}
	}

	go func() {
		for {
			StartScan()
			time.Sleep(time.Hour * 4)
		}
	}()

	// Service bits
	logger := serviceLog.NewLogfmtLogger(os.Stdout)
	logger = serviceLog.With(logger, "stamp", serviceLog.Timestamp(time.Now))

	var svc IpAllocator
	svc = ipAlloc{}
	svc = loggingMiddleware{logger, svc}

	reserveHandler := transport.NewServer(
		makeReserveEndpoint(svc),
		decodeReserveRequest,
		encodeResponse,
	)

	releaseHandler := transport.NewServer(
		makeReleaseEndpoint(svc),
		decodeReleaseRequest,
		encodeResponse,
	)

	http.Handle("/reserve", reserveHandler)
	http.Handle("/release", releaseHandler)

	logger.Log("err", http.ListenAndServe(":8080", nil))
}
