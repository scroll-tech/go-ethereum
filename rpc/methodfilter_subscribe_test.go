package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/scroll-tech/go-ethereum/log"
)

// TestMethodFilterSubscriptionSuffixBypass pins that a method entry cannot be
// used to smuggle in a subscription.
//
// A name ending in _subscribe is routed to handleSubscribe, which takes the
// namespace from the first segment of the method name and the subscription name
// from the first parameter. Neither is the name the filter matched on, so an
// entry such as "nftest:foo_subscribe" would otherwise allow every subscription
// of the namespace.
func TestMethodFilterSubscriptionSuffixBypass(t *testing.T) {
	log.Root().SetHandler(log.DiscardHandler())

	for _, entry := range []string{
		"nftest:foo_subscribe",
		"nftest:subscribe_subscribe",
		"nftest:foo_unsubscribe",
	} {
		if _, _, err := ParseAPIEntries([]string{entry}); err == nil {
			t.Errorf("entry %q was accepted; it routes to handleSubscribe and opens the namespace", entry)
		}
	}
}

// TestMethodFilterSubscriptionNotReachable drives the bypass end to end over an
// in-process connection: with only a plain method allowed, no subscription of
// the namespace may be started.
func TestMethodFilterSubscriptionNotReachable(t *testing.T) {
	log.Root().SetHandler(log.DiscardHandler())

	srv := filteredTestServer(t, "nftest:echo")
	client := DialInProc(srv)
	defer client.Close()

	// nftest_subscribe is not listed, so it must be refused.
	_, err := client.Subscribe(context.Background(), "nftest", make(chan interface{}), "someSubscription", 1, 1)
	if err == nil {
		t.Fatal("nftest_subscribe succeeded while unlisted")
	}
	if !strings.Contains(err.Error(), "does not exist/is not available") {
		t.Errorf("nftest_subscribe error = %q, want method-not-found", err)
	}
}
