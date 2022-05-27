# asynctask

这是一个给php设计的异步任务调度管理器.

首先需要把需要异步执行的任务写成一个可以访问的地址。

不同的地址之间调度是无序的，同一个地址不同参数的任务是有顺序的。

如果某种任务占用的资源过多，调度器会给这种任务降权，这样保证不会因为某个很慢的任务导至所有的异步任务都卡住。

简单说这个程序所做的就是将url地址从redis中取出然后按一定的优先级依次请求这些url

## 为什么这么做
可以按一般php的写法来做异步任务，不需要考虑cli环境php需要注意的那些问题。如：
* 不需要关心内存泄漏问题
* 不需要关心db/redis等连接超时的问题
* 可以容忍部分php代码有异常，不影响其它正常的代码
* 代码更新即时生效(与一般php一样)
* 运维工作比较简单(只要启动调度器)，其它都是一般的php服务
* 可以直接使用nginx等负载均衡功能！！

## 使用帮助

程序分为: http 模式 和 cmd 模式

自带帮助中 [ENV] 表示可以从环境变量中得到默认值

## 接口

接口      | 说明
----------| -------
/         | 服务运行状态监控页面
/status   | 监控状态json数据
/task/add | 添加任务接口

参数: id 任务id, name 任务, params 任务参数, parallel 并发数


## redis接口
可以配置好redis相关设置, 程序会从redis list中获取任务.

任务要求用json格式:`{id:任务id, parallel:并发数, name:任务(必需), params:[] 参数, add_time: 添加时间}`

* http模式参数示例: params:["a=1&b=2&c=3"]。需要url编码
* 推荐使用redis来添加任务

## 监控页截图

![监控页截图](https://raw.githubusercontent.com/bybzmt/asynctask/master/testtools/screenshot.gif)
