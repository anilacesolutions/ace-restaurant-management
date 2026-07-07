// Topic structure for the realtime layer. Build topic strings ONLY through
// these helpers — never concatenate by hand. Topic typos are silent failures.
//
//	restaurant/{rid}/table/{n}/orders          waiter publishes orders to a table
//	restaurant/{rid}/kitchen/print             backend → printer bridge
//	restaurant/{rid}/pos/charge                backend → POS bridge (Phase 2)
//	restaurant/{rid}/order/{oid}/status        cashier publishes status changes
//	restaurant/{rid}/presence/waiter/{wid}     LWT-driven presence (online/offline)
//	restaurant/{rid}/presence/bridge/{kind}    bridge presence (kitchen / pos)
//
// QoS policy:
//   - kitchen/print     : QoS 2 — must arrive exactly once; missing print = lost order
//   - pos/charge        : QoS 2 — financial; never duplicate, never lose
//   - table/{n}/orders  : QoS 1 — at-least-once; idempotent at the consumer (orderId dedup)
//   - order/{oid}/status: QoS 1
//   - presence/*        : QoS 1, retained, LWT
package mqttx

import "fmt"

const (
	QoSAtMostOnce  byte = 0
	QoSAtLeastOnce byte = 1
	QoSExactlyOnce byte = 2
)

func TableOrders(restaurantID string, tableNo int) string {
	return fmt.Sprintf("restaurant/%s/table/%d/orders", restaurantID, tableNo)
}

func TableOrdersAll(restaurantID string) string {
	return fmt.Sprintf("restaurant/%s/table/+/orders", restaurantID)
}

func KitchenPrint(restaurantID string) string {
	return fmt.Sprintf("restaurant/%s/kitchen/print", restaurantID)
}

func POSCharge(restaurantID string) string {
	return fmt.Sprintf("restaurant/%s/pos/charge", restaurantID)
}

func OrderStatus(restaurantID, orderID string) string {
	return fmt.Sprintf("restaurant/%s/order/%s/status", restaurantID, orderID)
}

func OrderStatusAll(restaurantID string) string {
	return fmt.Sprintf("restaurant/%s/order/+/status", restaurantID)
}

func WaiterPresence(restaurantID, waiterID string) string {
	return fmt.Sprintf("restaurant/%s/presence/waiter/%s", restaurantID, waiterID)
}

func BridgePresence(restaurantID, kind string) string {
	return fmt.Sprintf("restaurant/%s/presence/bridge/%s", restaurantID, kind)
}
