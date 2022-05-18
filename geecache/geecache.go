/*
 *Group 是 GeeCache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程
 *                            是
 * 接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
 *               |  否                         是
 *               |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
 *                           |  否
 *                           |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
 */
package geecache

import (
	"fmt"
	"log"
	"sync"
)
type Getter interface {										//定义接口Getter
	Get(key string) ([]byte, error)							//定义回调函数
}

type GetterFunc func (key string) ([]byte, error)			//定义函数类型GetterFunc

/*
 *函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
 */
func (f GetterFunc) Get(key string) ([]byte, error)  {		//实现Getter接口的Get方法
	return f(key)
}



/* 一个Group可以看作是一个缓存的命名空间 */
type Group struct {
	name		string			//每个Group都有一个唯一的名称。比如可以创建两个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info
	getter		Getter			//缓存未命中时获取源数据的回调(callback)
	mainCache	cache			//实现的并发缓存
}

var (
	mu sync.RWMutex
	groups = make(map[string]*Group)
)

/* 实例化Group。并将group存储在全局变量groups中 */
func NewGroup(name string, cacheBytes int64, getter Getter) *Group  {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:		name,
		getter:		getter,
		mainCache:	cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

/* 核心方法：Get, 实现了上述流程（1）（3）
 * 流程（1）：从mainCache 中查找缓存，如果存在则返回缓存值
 * 流程（2）：缓存不存在，则调用 load 方法，load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），
 * getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
 */
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b:	cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView)  {
	g.mainCache.add(key, value)
}