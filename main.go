<!DOCTYPE html>
<html>
<head>
<title>Sales – Ayan Watch and Electronics</title>

<style>
body{font-family:Arial;background:#eef2f3;padding:20px;}
.container{max-width:1200px;margin:auto;background:white;padding:20px;border-radius:10px;}
.summary{display:flex;gap:20px;margin:20px 0;}
.box{flex:1;background:#f1f7ff;padding:20px;text-align:center;font-weight:bold;border-radius:10px;}
table{width:100%;border-collapse:collapse;}
th,td{border:1px solid #ddd;padding:8px;text-align:center;}
th{background:#1f7ae0;color:white;}
button{padding:6px 12px;cursor:pointer;}
.delete-btn{background:red;color:white;border:none;}
.print-btn{background:#007bff;color:white;border:none;}
</style>
</head>

<body>

<div class="container">

<h2>Today’s Sales Report</h2>

<div class="summary">
<div class="box">Total Sales Count<br><span id="count">0</span></div>
<div class="box">Total Sales Amount ₹<br><span id="amount">0</span></div>
</div>

<table>
<thead>
<tr>
<th>ID</th>
<th>Customer</th>
<th>Product</th>
<th>Qty</th>
<th>Price</th>
<th>Payment</th>
<th>Date & Time</th>
<th>Print</th>
<th>Delete</th>
</tr>
</thead>
<tbody id="salesBody"></tbody>
</table>

</div>

<script>

function formatIST(dateStr){
    return new Date(dateStr).toLocaleString("en-IN",{timeZone:"Asia/Kolkata"});
}

async function loadSales(){

const res=await fetch("/sales");
const data=await res.json();

const body=document.getElementById("salesBody");
body.innerHTML="";

let count=0;
let amount=0;

const today=new Date().toLocaleDateString("en-CA",{timeZone:"Asia/Kolkata"});

data.forEach(s=>{

if(!s.createdDate) return;

const saleDate=new Date(s.createdDate).toLocaleDateString("en-CA",{timeZone:"Asia/Kolkata"});

if(saleDate===today){

count++;
amount+=s.price*s.quantity;

body.innerHTML+=`
<tr>
<td>${s.saleId}</td>
<td>${s.customerName}</td>
<td>${s.productName}</td>
<td>${s.quantity}</td>
<td>₹${s.price}</td>
<td>${s.paymentMethod}</td>
<td>${formatIST(s.createdDate)}</td>
<td><button class="print-btn" onclick="printBill(${s.saleId})">Print</button></td>
<td><button class="delete-btn" onclick="deleteSale(${s.saleId})">Delete</button></td>
</tr>`;
}

});

document.getElementById("count").innerText=count;
document.getElementById("amount").innerText=amount;

}

function printBill(id){
window.print();
}

async function deleteSale(id){

if(!confirm("Delete this sale?")) return;

await fetch("/sales/delete",{
method:"POST",
headers:{"Content-Type":"application/json"},
body:JSON.stringify({saleId:id})
});

loadSales();
}

document.addEventListener("DOMContentLoaded",loadSales);

</script>

</body>
</html>
