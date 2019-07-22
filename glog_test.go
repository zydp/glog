package glog

import (
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	/* default 100,10 (the logfile file size 100MB, total split 10 times)*/
	var logger = New("test.log", "[Info] ", Ldate | Ltime | Lshortfile)
	logger.Println("Hello!")
	logger.SetPrefix("[ChanePrefix]")
	logger.Println("Glog!")
	var Wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		Wg.Add(1)
		go func(count int){
			for i := 0; i < count; i++ {
				logger.Printf("%s-%d", "abcdefghijklmnopqrstuvwxyz", 123456789)
				logger.Println("abcdefghijklmnopqrstuvwxyz0123456789你好，我是测试日志~!@#$%^&*()_+{}|:")
			}
			Wg.Done()
		}(100000)
	}
	Wg.Wait()

}

func TestNewEx(t *testing.T) {
	var logger = NewEx("test.log", "[Info] ", Ldate | Ltime | Lshortfile, 10, 5)
	logger.Println("Hello!")
	logger.SetPrefix("[ChanePrefix]")
	logger.Println("Glog!")

	var Wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		Wg.Add(1)
		go func(count int){
			for i := 0; i < count; i++ {
				logger.Printf("%s-%d", "abcdefghijklmnopqrstuvwxyz", 123456789)
				logger.Println("abcdefghijklmnopqrstuvwxyz0123456789你好，我是测试日志~!@#$%^&*()_+{}|:")
			}
			Wg.Done()
		}(10000)
	}
	Wg.Wait()
}