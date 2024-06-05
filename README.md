# asynctask

这是一个给php设计的异步任务调度管理器.

首先需要把需要异步执行的任务写成一个可以访问的地址或cmd。

任务是并发执行的,需要顺序执行的任务必需指定并发为1。

如果某种任务占用的资源过多，调度器会给这种任务降权，这样保证不会因为某个很慢的任务导至所有的异步任务都卡住。

简单说这个程序所做的就是将任务从redis中取出然后按一定的优先级任行这些任务

## 为什么这么做
可以按一般php的写法来做异步任务，不需要考虑常驻cli环境php需要注意的那些问题。如：
* 不需要关心内存泄漏问题
* 不需要关心db/redis等连接超时的问题
* 可以容忍php代码有异常，不影响其它正常的代码
* 代码更新即时生效
* 运维工作比较简单(只要启动调度器)，其它都是一般的php服务
* 可以直接使用nginx等负载均衡功能

## 使用帮助

任务使用json格式
通过 http接口 /api/task/add
或 redis队列添加

字段      | 类型              | 默认值 | 说明
----------| ----------------- | ------ | ------------
method    | string            | GET    | 服务运行状态监控页面
url       | string            | 必选!  | 监控状态json数据
header    | map[string]string | 无     | 添加任务接口
body      | []byte            | 无     | POST时请求体，要base64编码
runat     | int               | 0      | 任务执行时间, 单位秒
timeout   | int               | 0      | 任务执行超时间
hold      | string            | 无     | 原样输出在日志方便排错
status    | int               | 0      | 任务确定响应结果, 0为自动
retry     | int               | 0      | 出错时重试次数
interval  | interval          | 1      | 重试间隔,单位秒


## 配置


```
type Config struct {
	Group       string //默认组
	WorkerNum   uint32 //默认工作线程数量
	Parallel    uint32 //默认并发数
	Timeout     uint   //默认超时
	JobsMaxIdle uint   //空闲数量
	CloseWait   uint   //关闭等待
	HttpAddr    string //http监听端口
	HttpEnable  bool   //http是否开启

	Jobs   []*Job              //任务配置
	Routes []*Route            //路由配置
	Groups map[string]*Group   //任务组配置
	Dirver map[string]*Dirver  //驱动配置
	Redis  []RedisConfig       //redis队列
	Crons  []CronTask          //计化任务
}

type Job struct {
	Pattern  string //正则，命中的会使用这条配置
	Group    string //任务组
	Priority int32  //权重系数
	Parallel uint32 //并发数
}
```




