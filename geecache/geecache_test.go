package geecache

import (
	"testing"
	"fmt"
	"reflect"
	"log"
)
/* 利用map模拟耗时的数据库 */
var db = map[string] string {
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

/* 写一个测试用例，用来保证回调函数能够正常工作 */
func TestGetter(t *testing.T)  {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {		//借助 GetterFunc 的类型转换，将一个匿名回调函数转换成了接口 f Getter
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {			//调用该接口的方法 f.Get(key string)，实际上就是在调用匿名回调函数
		t.Errorf("callback failed")
	}
}

/* 创建group实例，并测试Get方法 */
func TestGet(t *testing.T)  {
	loadCounts := make(map[string]int, len(db))
	gee := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	
	//缓存为空时，调用了回调函数，第二次访问时，则直接从缓存中读取
	for k, v := range db {
		if view, err := gee.Get(k); err != nil || view.String() != v {		//在缓存为空的情况下，能够通过回调函数获取到源数据
			t.Fatal("failed to get value of Tom")
		}	// load from callback function
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {			//在缓存已经存在的情况下，是否直接从缓存中获取，为了实现这一点，使用 loadCounts 统计某个键调用回调函数的次数，如果次数大于1，则表示调用了多次回调函数，没有缓存
			t.Fatalf("cache %s miss", k)
		}	// cache hit

	}
	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unknown should be empty, but %s got", view)
	}
}