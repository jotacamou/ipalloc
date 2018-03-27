package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"gopkg.in/mgo.v2/bson"
)

// IpAllocator provides operations on IP inventory.
type IpAllocator interface {
	Reserve(string) (string, error)
	Release(string) (string, error)
}

type ipAlloc struct{}

type loggingMiddleware struct {
	logger log.Logger
	next   IpAllocator
}

func (mw loggingMiddleware) Reserve(vlan string) (output string, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "reserve",
			"input", vlan,
			"output", output,
			"err", err,
			"time", time.Since(begin),
		)
	}(time.Now())
	output, err = mw.next.Reserve(vlan)
	return
}

func (mw loggingMiddleware) Release(ip string) (output string, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "release",
			"input", ip,
			"output", output,
			"err", err,
			"time", time.Since(begin),
		)
	}(time.Now())
	output, err = mw.next.Release(ip)
	return
}

func (ipAlloc) Reserve(vlan string) (string, error) {
	c := GetCollection()
	var result IP
	query := bson.M{
		"name":     vlan,
		"alive":    false,
		"lock":     false,
		"reserved": false,
	}
	err := c.Find(query).One(&result)
	if err != nil {
		return "", err
	}
	query = bson.M{"_id": result.Id}
	reserve := bson.M{"$set": bson.M{"reserved": true}}
	err = c.Update(query, reserve)
	if err != nil {
		return "", err
	}
	return result.Addr, nil
}

func (ipAlloc) Release(ip string) (string, error) {
	if net.ParseIP(ip) == nil {
		return "", errors.New("invalid ip")
	}
	id := ip2int(ip)
	c := GetCollection()
	query := bson.M{"_id": id}
	release := bson.M{"$set": bson.M{"reserved": false}}
	err := c.Update(query, release)
	if err != nil {
		return "", err
	}
	return ip + " released", nil
}

func makeReserveEndpoint(svc IpAllocator) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(reserveRequest)
		v, err := svc.Reserve(req.Vlan)
		if err != nil {
			return reserveResponse{v, err.Error()}, nil
		}
		return reserveResponse{v, ""}, nil
	}
}

func makeReleaseEndpoint(svc IpAllocator) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(releaseRequest)
		v, err := svc.Release(req.Ip)
		if err != nil {
			return reserveResponse{v, err.Error()}, nil
		}
		return releaseResponse{v, ""}, nil
	}
}

func decodeReserveRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request reserveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeReleaseRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request releaseRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

type reserveRequest struct {
	Vlan string `json:"vlan"`
}

type reserveResponse struct {
	V   string `json:"ipaddr"`
	Err string `json:"err,omitempty"`
}

type releaseRequest struct {
	Ip string `json:"ip"`
}

type releaseResponse struct {
	V   string `json:"msg"`
	Err string `json:"err,omitempty"`
}
