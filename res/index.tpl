{{define "body"}}

<div>异步任务状态</div>
<div id="All">
</div>
<div id="jobs">
</div>

<script>

function showStatus() {
    $.getJSON("/status", function(json){
        html = '<table>';
        html += '<tr>';
        html += '<th>名称</th>';
        html += '<th>负载</th>';
        html += '<th>执行中</th>';
        html += '<th>己执行</th>';
        html += '<th>队列</th>';
        html += '</tr>';
        html += '</table>';
        $("#All").html(html);

        j = json.Data.All;
        html = "<tr>"
        html += '<td class="name">总体</td>';
        html += '<td class="load">'+(j.Load/100)+'%</td>';
        html += '<td class="now">'+j.NowNum+'</td>';
        html += '<td class="run">'+j.RunNum+'</td>';
        html += '<td class="wait">'+j.WaitNum+'</td>';
        $("#All table").append(html);

        html = '<table>';
        html += '<tr>';
        html += '<th>名称</th>';
        html += '<th>占比</th>';
        html += '<th>执行中</th>';
        html += '<th>己执行</th>';
        html += '<th>队列</th>';
        html += '</tr>';
        html += '</table>';
        $("#jobs").html(html);

        for (i in json.Data.Jobs) {
            j = json.Data.Jobs[i];
            html = "<tr>"
            html += '<td class="name">'+i+'</td>';
            html += '<td class="load">'+(j.Load/100)+'%</td>';
            html += '<td class="now">'+j.NowNum+'</td>';
            html += '<td class="run">'+j.RunNum+'</td>';
            html += '<td class="wait">'+j.WaitNum+'</td>';
            html += "</tr>"
            $("#jobs table").append(html);
        }
    });
}

window.onload = function(){
    showStatus();
};

window.setInterval(function(){
    showStatus();
});
</script>

{{end}}
