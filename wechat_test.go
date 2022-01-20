package wechat

import (
	"fmt"
	"github.com/gotid/wechat/context"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	wc1 := Get(&context.Context{AppID: "1"})
	fmt.Printf("%p \n", &wc1)
	time.Sleep(200 * time.Millisecond)

	wc2 := Get(&context.Context{AppID: "1"})
	fmt.Printf("%p \n", &wc2)
}
