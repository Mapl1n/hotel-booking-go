package router

import "github.com/gin-gonic/gin"

func serveHomePage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, homePageHTML)
}

const homePageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>智慧酒店管理系统</title>
<style>
:root {
  --bg: #0f172a; --card: #1e293b; --border: #334155;
  --text: #e2e8f0; --muted: #94a3b8; --accent: #3b82f6;
  --green: #22c55e; --red: #ef4444; --orange: #f59e0b;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: var(--bg); color: var(--text); min-height: 100vh; }
.header { background: var(--card); border-bottom: 1px solid var(--border); padding: 14px 24px; display: flex; justify-content: space-between; align-items: center; position: sticky; top: 0; z-index: 100; }
.header h1 { font-size: 18px; }
.header .user { font-size: 13px; color: var(--muted); }
.container { max-width: 1100px; margin: 0 auto; padding: 20px; }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 20px; margin-bottom: 16px; }
.card h3 { font-size: 15px; margin-bottom: 14px; color: var(--accent); }
.btn { padding: 8px 18px; border-radius: 6px; border: none; cursor: pointer; font-size: 13px; font-weight: 500; transition: all .2s; }
.btn-primary { background: var(--accent); color: #fff; }
.btn-primary:hover { background: #2563eb; }
.btn-green { background: var(--green); color: #fff; }
.btn-red { background: var(--red); color: #fff; }
.btn-outline { background: transparent; border: 1px solid var(--border); color: var(--text); }
.btn-outline:hover { background: var(--border); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.input-group { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 10px; }
input, select { background: var(--bg); border: 1px solid var(--border); color: var(--text); padding: 8px 12px; border-radius: 6px; font-size: 13px; outline: none; }
input:focus, select:focus { border-color: var(--accent); }
input { flex: 1; min-width: 140px; }
label { font-size: 12px; color: var(--muted); display: block; margin-bottom: 4px; }
.form-group { display: flex; flex-direction: column; gap: 2px; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; padding: 10px 8px; border-bottom: 2px solid var(--border); color: var(--muted); font-weight: 500; font-size: 12px; }
td { padding: 10px 8px; border-bottom: 1px solid var(--border); }
tr:hover td { background: rgba(59,130,246,0.05); }
.badge { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 11px; }
.badge-pending { background: var(--orange); color: #000; }
.badge-paid { background: var(--accent); color: #fff; }
.badge-checked_in { background: var(--green); color: #000; }
.badge-cancelled { background: var(--red); color: #fff; }
.badge-available { background: #064e3b; color: #6ee7b7; }
.badge-occupied { background: #7f1d1d; color: #fca5a5; }
.price { color: #fbbf24; font-weight: 600; }
.tabs { display: flex; gap: 4px; margin-bottom: 16px; flex-wrap: wrap; }
.tab { padding: 8px 16px; border-radius: 6px; cursor: pointer; font-size: 13px; background: var(--bg); border: 1px solid var(--border); color: var(--muted); transition: all .2s; }
.tab.active { background: var(--accent); color: #fff; border-color: var(--accent); }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }
.room-card { padding: 14px; border: 1px solid var(--border); border-radius: 8px; cursor: pointer; transition: all .2s; }
.room-card:hover { border-color: var(--accent); background: rgba(59,130,246,0.08); }
.room-card .rn { font-size: 20px; font-weight: 700; color: var(--accent); }
.room-card .rt { font-size: 13px; color: var(--muted); margin: 4px 0; }
.room-card .pr { font-size: 16px; color: #fbbf24; font-weight: 600; }
#toast { position: fixed; top: 20px; right: 20px; z-index: 9999; display: flex; flex-direction: column; gap: 8px; }
.toast-msg { padding: 12px 20px; border-radius: 8px; font-size: 13px; animation: slideIn .3s ease; max-width: 380px; }
.toast-success { background: #065f46; color: #6ee7b7; border: 1px solid #059669; }
.toast-error { background: #7f1d1d; color: #fca5a5; border: 1px solid #dc2626; }
@keyframes slideIn { from { transform: translateX(100%); opacity: 0; } to { transform: translateX(0); opacity: 1; } }
@media (max-width: 600px) { .input-group { flex-direction: column; } .header { flex-direction: column; gap: 8px; } }
</style>
</head>
<body>

<div class="header">
  <h1>🏨 智慧酒店管理系统</h1>
  <div style="display:flex;align-items:center;gap:12px">
    <span class="user" id="userInfo">未登录</span>
    <button class="btn btn-outline btn-sm" id="loginBtn" onclick="showLogin()">登录</button>
  </div>
</div>

<div id="toast"></div>

<div class="container" id="app">
  <!-- Login -->
  <div class="card" id="loginCard" style="max-width:420px;margin:40px auto">
    <h3>🔑 登录</h3>
    <div class="input-group"><input id="loginUser" placeholder="手机号/admin" value="admin"></div>
    <div class="input-group"><input id="loginPass" type="password" placeholder="密码" value="123456"></div>
    <button class="btn btn-primary" onclick="doLogin()" style="width:100%">登录</button>
    <p style="margin-top:12px;font-size:12px;color:var(--muted)">演示: admin / 13800008888 | 密码: 123456</p>
  </div>

  <!-- Main (hidden until login) -->
  <div id="mainContent" style="display:none">
    <div class="tabs" id="tabs">
      <div class="tab active" onclick="switchTab('hotels')">🏨 酒店</div>
      <div class="tab" onclick="switchTab('rooms')">🚪 空房查询</div>
      <div class="tab" onclick="switchTab('orders')">📋 我的订单</div>
    </div>
    <div id="tabContent"></div>
  </div>
</div>

<script>
const API = '/api';
let token = '';
let user = null;

function toast(msg, type) {
  const el = document.getElementById('toast');
  const d = document.createElement('div');
  d.className = 'toast-msg ' + (type === 'error' ? 'toast-error' : 'toast-success');
  d.textContent = msg;
  el.appendChild(d);
  setTimeout(() => d.remove(), 3000);
}

async function api(method, path, body) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = 'Bearer ' + token;
  const opts = { method, headers };
  if (body) opts.body = JSON.stringify(body);
  const r = await fetch(API + path, opts);
  const d = await r.json();
  if (d.code !== 0 && d.code !== 200) throw new Error(d.message || 'Error');
  return d.data;
}

async function doLogin() {
  try {
    const data = await api('POST', '/auth/login', {
      username: document.getElementById('loginUser').value,
      password: document.getElementById('loginPass').value,
    });
    token = data.access_token;
    user = data.user;
    document.getElementById('loginCard').style.display = 'none';
    document.getElementById('mainContent').style.display = 'block';
    document.getElementById('userInfo').textContent = user.username + ' (' + user.role + ')';
    document.getElementById('loginBtn').textContent = '退出';
    document.getElementById('loginBtn').onclick = () => { token = ''; user = null; location.reload(); };
    toast('登录成功！', 'success');
    switchTab('hotels');
  } catch (e) { toast(e.message, 'error'); }
}

function showLogin() {
  if (token) { token = ''; user = null; location.reload(); return; }
  document.getElementById('loginCard').style.display = 'block';
}

async function switchTab(tab) {
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  event.target.classList.add('active');
  const c = document.getElementById('tabContent');
  switch (tab) {
    case 'hotels': await loadHotels(c); break;
    case 'rooms': loadRoomSearch(c); break;
    case 'orders': await loadOrders(c); break;
  }
}

async function loadHotels(container) {
  try {
    const hotels = await api('GET', '/hotels');
    container.innerHTML = '<h3>🏨 酒店列表</h3><div class="grid">' +
      hotels.map(h => '<div class="card" style="cursor:pointer" onclick="viewHotel(' + h.id + ')"><h4>' + h.name + '</h4><p style="font-size:13px;color:var(--muted);margin-top:6px">📍 ' + (h.address || '暂无地址') + '</p><p style="font-size:13px;color:var(--muted)">📞 ' + (h.phone || '暂无电话') + '</p></div>').join('') +
      '</div>';
  } catch (e) { container.innerHTML = '<p style="color:var(--red)">加载失败: ' + e.message + '</p>'; }
}

async function viewHotel(hotelID) {
  const container = document.getElementById('tabContent');
  try {
    const [types, h] = await Promise.all([
      api('GET', '/rooms/types?hotel_id=' + hotelID),
      api('GET', '/hotels/' + hotelID)
    ]);
    container.innerHTML =
      '<h3>🏨 ' + h.name + '</h3>' +
      '<p style="font-size:13px;color:var(--muted);margin-bottom:12px">📍 ' + (h.address||'') + ' | 📞 ' + (h.phone||'') + '</p>' +
      '<h4 style="margin-bottom:8px">房型 & 价格</h4><div class="grid">' +
      types.map(t => '<div class="card"><h4>' + t.name + '</h4><p class="price" style="font-size:24px;margin:8px 0">¥' + t.price + '<span style="font-size:13px;color:var(--muted)">/晚</span></p><p style="font-size:12px;color:var(--muted)">' + (t.description||'') + ' | 可住' + t.capacity + '人</p></div>').join('') +
      '</div>' +
      '<button class="btn btn-outline" style="margin-top:12px" onclick="switchTab(\'rooms\')">查空房 →</button>';
  } catch (e) { container.innerHTML = '<p style="color:var(--red)">加载失败</p>'; }
}

function loadRoomSearch(container) {
  container.innerHTML =
    '<h3>🚪 查询可用房间</h3>' +
    '<div class="card">' +
    '<div class="input-group">' +
    '<div class="form-group"><label>酒店ID</label><input id="rhotel" value="1" style="width:80px"></div>' +
    '<div class="form-group"><label>入住日期</label><input id="rcheckin" type="date" value="2026-07-20"></div>' +
    '<div class="form-group"><label>退房日期</label><input id="rcheckout" type="date" value="2026-07-23"></div>' +
    '<div class="form-group"><label>&nbsp;</label><button class="btn btn-primary" onclick="searchRooms()">🔍 搜索</button></div>' +
    '</div></div><div id="roomResults"></div>';
}

async function searchRooms() {
  const hotelID = document.getElementById('rhotel').value;
  const checkIn = document.getElementById('rcheckin').value;
  const checkOut = document.getElementById('rcheckout').value;
  const container = document.getElementById('roomResults');
  try {
    const rooms = await api('GET', '/rooms?hotel_id=' + hotelID + '&check_in=' + checkIn + '&check_out=' + checkOut);
    if (!rooms || rooms.length === 0) {
      container.innerHTML = '<p style="color:var(--muted)">该时段暂无可用房间</p>';
      return;
    }
    container.innerHTML = '<h4 style="margin:12px 0 8px">找到 ' + rooms.length + ' 间可用房间</h4><div class="grid">' +
      rooms.map(r => {
        const rt = r.room_type || {};
        return '<div class="room-card" onclick="bookRoom(' + r.id + ',\'' + r.room_number + '\',' + rt.price + ',\'' + rt.name + '\')"><div class="rn">' + r.room_number + '</div><div class="rt">' + rt.name + '</div><div class="pr">¥' + rt.price + '/晚</div><div style="font-size:11px;color:var(--muted);margin-top:4px">' + r.floor + '楼 | 可住' + (rt.capacity||2) + '人</div></div>';
      }).join('') +
      '</div>';
  } catch (e) { container.innerHTML = '<p style="color:var(--red)">查询失败: ' + e.message + '</p>'; }
}

async function bookRoom(roomID, roomNumber, price, typeName) {
  const checkIn = document.getElementById('rcheckin').value;
  const checkOut = document.getElementById('rcheckout').value;
  const guestName = prompt('入住人姓名:', '张三');
  if (!guestName) return;
  const idCard = prompt('身份证号 (18位):', '330106199001011234');
  if (!idCard || idCard.length !== 18) { toast('身份证号需18位', 'error'); return; }

  try {
    const result = await api('POST', '/orders', {
      room_id: roomID, guest_name: guestName, id_card: idCard,
      check_in: checkIn, check_out: checkOut,
    });
    const nights = Math.round((new Date(checkOut) - new Date(checkIn)) / 86400000);
    toast('预订成功！订单号: ' + result.order_no + ' | ¥' + result.total_price + ' (' + nights + '晚)', 'success');
    searchRooms(); // refresh
  } catch (e) { toast(e.message, 'error'); }
}

async function loadOrders(container) {
  try {
    const orders = await api('GET', '/orders');
    if (!orders || orders.length === 0) {
      container.innerHTML = '<h3>📋 我的订单</h3><p style="color:var(--muted)">暂无订单</p>';
      return;
    }
    const statusBadge = s => '<span class="badge badge-' + s + '">' + ({pending:'待支付',paid:'已支付',checked_in:'已入住',checked_out:'已退房',cancelled:'已取消'}[s]||s) + '</span>';
    container.innerHTML =
      '<h3>📋 我的订单 (' + orders.length + ')</h3>' +
      '<table><thead><tr><th>订单号</th><th>房间</th><th>酒店</th><th>入住</th><th>退房</th><th>金额</th><th>状态</th><th>操作</th></tr></thead><tbody>' +
      orders.map(o => '<tr><td>' + (o.order_no || '#') + '</td><td>' + o.room_number + '</td><td>' + (o.hotel_name||'') + '</td><td>' + o.check_in + '</td><td>' + o.check_out + '</td><td class="price">¥' + o.total_price + '</td><td>' + statusBadge(o.status) + '</td><td>' +
        (o.status === 'pending'
          ? '<button class="btn btn-green btn-sm" onclick="payOrder(' + o.id + ')">💳 支付</button> '
          : '') +
        '<button class="btn btn-outline btn-sm" onclick="viewOrder(' + o.id + ')">详情</button>' +
      '</td></tr>').join('') +
      '</tbody></table>';
  } catch (e) { container.innerHTML = '<p style="color:var(--red)">加载失败: ' + e.message + '</p>'; }
}

async function payOrder(orderID) {
  try {
    const r = await api('POST', '/payment/create', { order_id: orderID, payment_method: 'mock' });
    toast('支付成功！流水号: ' + r.payment_no, 'success');
    switchTab('orders');
  } catch (e) { toast(e.message, 'error'); }
}

async function viewOrder(orderID) {
  try {
    const o = await api('GET', '/orders/' + orderID);
    const rooms = o.room || {};
    const rt = rooms.room_type || {};
    const container = document.getElementById('tabContent');
    container.innerHTML =
      '<h3>📋 订单详情 #' + o.order_no + '</h3>' +
      '<div class="card"><table>' +
      '<tr><td style="color:var(--muted)">订单状态</td><td>' + o.status + '</td></tr>' +
      '<tr><td style="color:var(--muted)">入住人</td><td>' + o.guest_name + '</td></tr>' +
      '<tr><td style="color:var(--muted)">身份证</td><td>' + o.id_card_masked + '</td></tr>' +
      '<tr><td style="color:var(--muted)">房间</td><td>' + rooms.room_number + ' (' + rt.name + ') ' + rooms.floor + '楼</td></tr>' +
      '<tr><td style="color:var(--muted)">入住/退房</td><td>' + o.check_in + ' → ' + o.check_out + '</td></tr>' +
      '<tr><td style="color:var(--muted)">总价</td><td class="price">¥' + o.total_price + '</td></tr>' +
      '<tr><td style="color:var(--muted)">下单时间</td><td>' + o.created_at + '</td></tr>' +
      '</table></div>' +
      '<button class="btn btn-outline" onclick="switchTab(\'orders\')">← 返回列表</button>';
  } catch (e) { toast(e.message, 'error'); }
}
</script>
</body></html>`
