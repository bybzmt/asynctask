<!DOCTYPE html>
<html>
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
    <script src="/res/jquery-3.2.1.min.js"></script>
    <style>
body{
    padding:0;
    margin:5px;
}

.table {
    width:100%;
    border-collapse:collapse;
}
.table td, .table th{
    border:1px solid #777;
}
.th1{
    width:200px;
}
.th4{
    width:100px;
}
.th5{
    width:150px;
}

.current .name{
    color:green;
}

.confirm_msg{
    text-align:center;
    font-size:24px;
}
.confirm_button{
    text-align:center;
    font-size:24px;
}
.confirm_button a{
    color:#000;
    text-decoration:none;
}

.button{
    text-align:center;
}
.button a{
    color:#000;
    text-decoration:none;
}

    </style>
</head>
<body>
	{{template "body" $}}
</body>
</html>
