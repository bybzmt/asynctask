{{define "body"}}

<div>异步任务状态</div>

<div style="width:500px">
    <div id="All" style="float:left;">
        <table>
            <thead>
                <tr>
                   <th>名称</th>
                   <th>负载</th>
                   <th>执行中</th>
                   <th>己执行</th>
                   <th>昨天</th>
                   <th>队列</th>
               </tr>
            </thead>
            <tbody>
            </tbody>
        </table>
    </div>

    <div style="float:right;">
    <label for="sortby">排序:</label>
    <select id="sortby">
        <option value="0">名称正序</option>
        <option value="1">名称倒序</option>
        <option selected value="2">占用</option>
        <option value="3">执行中</option>
        <option value="4">己执行</option>
        <option value="5">己执行</option>
        <option value="6">队列</option>
        <option value="7">平均时间</option>
    </select>
    </div>

    <div style="clear:both;"></div>
</div>

<div id="jobs">
    <table>
        <thead>
            <tr>
               <th>名称</th>
               <th>负载</th>
               <th>执行中</th>
               <th>己执行</th>
               <th>昨天</th>
               <th>队列</th>
                <th>平均时间</th>
           </tr>
        </thead>
        <tbody>
        </tbody>
    </table>
</div>

<script>

function showStatus() {
    $.getJSON("/status", function(json){
        j = json.Data.All;

        var html = "";

        html = "<tr>"
        html += '<td class="name">总体</td>';
        html += '<td class="load">'+(j.Load/100)+'%</td>';
        html += '<td class="now">'+j.NowNum+'</td>';
        html += '<td class="run">'+j.RunNum+'</td>';
        html += '<td class="old">'+j.OldNum+'</td>';
        html += '<td class="wait">'+j.WaitNum+'</td>';
        html += "</tr>"
        $("#All table tbody").html(html);

        var sortby = $("#sortby").val();

        json.Data.Jobs = json.Data.Jobs.sort(function(a, b){
            switch (sortby) {
            case "0": return a.Name.localeCompare(b.Name)
            case "1": return b.Name.localeCompare(a.Name)
            case "2": return b.Load - a.Load;
            case "3": return b.NowNum - a.NowNum;
            case "4": return b.RunNum - a.RunNum;
            case "5": return b.OldNum - a.OldNum;
            case "6": return b.WaitNum - a.WaitNum;
            case "7": return b.UseTime - a.UseTime;
            }
        });

        html = "";
        for (var i=0; i<json.Data.Jobs.length; i++) {
            j = json.Data.Jobs[i];
            html += "<tr>"
            html += '<td class="name">'+j.Name+'</td>';
            html += '<td class="load">'+(j.Load/100)+'%</td>';
            html += '<td class="now">'+j.NowNum+'</td>';
            html += '<td class="run">'+j.RunNum+'</td>';
            html += '<td class="old">'+j.OldNum+'</td>';
            html += '<td class="wait">'+j.WaitNum+'</td>';
            html += '<td class="time">'+(j.UseTime/1000)+'s</td>';
            html += "</tr>"
        }
        $("#jobs table tbody").html(html);
    });
}

window.onload = function(){
    showStatus();
};

window.setInterval(function(){
    showStatus();
}, 500);

</script>

{{end}}
