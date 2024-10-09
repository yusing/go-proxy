package loadbalancer

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestRebalance(t *testing.T) {
	t.Parallel()
	t.Run("zero", func(t *testing.T) {
		lb := New(Config{})
		for range 10 {
			lb.AddServer(&Server{})
		}
		lb.Rebalance()
		ExpectEqual(t, lb.sumWeight, maxWeight)
	})
	t.Run("less", func(t *testing.T) {
		lb := New(Config{})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .1)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .2)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .3)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .2)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .1)})
		lb.Rebalance()
		// t.Logf("%s", U.Must(json.MarshalIndent(lb.pool, "", "  ")))
		ExpectEqual(t, lb.sumWeight, maxWeight)
	})
	t.Run("more", func(t *testing.T) {
		lb := New(Config{})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .1)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .2)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .3)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .4)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .3)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .2)})
		lb.AddServer(&Server{Weight: weightType(float64(maxWeight) * .1)})
		lb.Rebalance()
		// t.Logf("%s", U.Must(json.MarshalIndent(lb.pool, "", "  ")))
		ExpectEqual(t, lb.sumWeight, maxWeight)
	})
}
