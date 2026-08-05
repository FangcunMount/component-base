package nsq

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const lookupdResolveTimeout = 5 * time.Second

type lookupdProducer struct {
	BroadcastAddress string `json:"broadcast_address"`
	Hostname         string `json:"hostname"`
	TCPPort          int    `json:"tcp_port"`
}

type lookupdResponse struct {
	Producers []lookupdProducer `json:"producers"`
}

func resolveTopicProducers(ctx context.Context, lookupdAddrs []string, topic string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, lookupdResolveTimeout)
	defer cancel()

	type lookupResult struct {
		address   string
		producers []lookupdProducer
		err       error
	}
	results := make(chan lookupResult, len(lookupdAddrs))
	for _, lookupdAddr := range lookupdAddrs {
		lookupdAddr := lookupdAddr
		go func() {
			producers, err := queryLookupdProducers(ctx, lookupdAddr, topic)
			results <- lookupResult{address: lookupdAddr, producers: producers, err: err}
		}()
	}

	addresses := make(map[string]struct{})
	var failures []string
	for range lookupdAddrs {
		result := <-results
		if result.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", result.address, result.err))
			continue
		}
		for _, producer := range result.producers {
			host := strings.TrimSpace(producer.BroadcastAddress)
			if host == "" {
				host = strings.TrimSpace(producer.Hostname)
			}
			if host == "" || producer.TCPPort <= 0 {
				continue
			}
			addresses[net.JoinHostPort(host, strconv.Itoa(producer.TCPPort))] = struct{}{}
		}
	}
	if len(addresses) == 0 {
		if len(failures) > 0 {
			return nil, fmt.Errorf("no lookupd returned a producer (%s)", strings.Join(failures, "; "))
		}
		return nil, fmt.Errorf("no lookupd returned a producer")
	}

	result := make([]string, 0, len(addresses))
	for address := range addresses {
		result = append(result, address)
	}
	sort.Strings(result)
	return result, nil
}

func queryLookupdProducers(ctx context.Context, lookupdAddr, topic string) ([]lookupdProducer, error) {
	base := strings.TrimSpace(lookupdAddr)
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("parse lookupd address: %w", err)
	}
	endpoint.Path = "/lookup"
	query := endpoint.Query()
	query.Set("topic", topic)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build lookup request: %w", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("query lookupd: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("lookupd status %s", response.Status)
	}
	var payload lookupdResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode lookupd response: %w", err)
	}
	return payload.Producers, nil
}
