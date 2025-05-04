# استخدام MongoDB مع Evilginx2

تم إضافة دعم لاستخدام MongoDB كبديل لقاعدة البيانات المضمنة (BuntDB) في Evilginx2. يسمح هذا بتخزين أكثر استقرارًا وقابلية للتوسع، وإمكانية استعلام أفضل عن البيانات، وسهولة النسخ الاحتياطي واستعادة البيانات.

## المتطلبات

- تثبيت خادم MongoDB (محلي أو عن بعد)
- إمكانية الوصول إلى خادم MongoDB من الجهاز الذي يشغل Evilginx2

## الإعداد

### 1. تثبيت MongoDB

#### على نظام Ubuntu/Debian:

```bash
# إضافة مفتاح MongoDB GPG
wget -qO - https://www.mongodb.org/static/pgp/server-6.0.asc | sudo apt-key add -

# إضافة مستودع MongoDB
echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/6.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-6.0.list

# تحديث المستودعات وتثبيت MongoDB
sudo apt-get update
sudo apt-get install -y mongodb-org

# تشغيل خدمة MongoDB
sudo systemctl start mongod
sudo systemctl enable mongod
```

#### على نظام macOS (باستخدام Homebrew):

```bash
brew tap mongodb/brew
brew install mongodb-community
brew services start mongodb-community
```

### 2. إنشاء قاعدة بيانات وإضافة مستخدم (اختياري ولكن موصى به)

```bash
# الاتصال بـ MongoDB
mongosh

# إنشاء قاعدة بيانات
use evilginx

# إنشاء مستخدم مع صلاحيات
db.createUser({
  user: "evilginx_user",
  pwd: "evilginx_password",
  roles: [{ role: "readWrite", db: "evilginx" }]
})

# الخروج
exit
```

## استخدام Evilginx2 مع MongoDB

### خيارات سطر الأوامر الإضافية:

```
--use-mongo          استخدام MongoDB بدلاً من قاعدة البيانات المضمنة (BuntDB)
--mongo-uri string   عنوان اتصال MongoDB (افتراضي: "mongodb://localhost:27017")
--mongo-db string    اسم قاعدة بيانات MongoDB (افتراضي: "evilginx")
```

### مثال لتشغيل Evilginx2 مع MongoDB محلي:

```bash
./evilginx2 --use-mongo
```

### مثال لتشغيل Evilginx2 مع MongoDB بعيد ومصادقة:

```bash
./evilginx2 --use-mongo --mongo-uri="mongodb://evilginx_user:evilginx_password@example.com:27017/?authSource=evilginx" --mongo-db="evilginx"
```

## ميزات استخدام MongoDB

1. **إمكانية الاستعلام المتقدمة**: يمكن استخدام أدوات مثل MongoDB Compass أو Robo 3T للاستعلام عن البيانات بشكل مرئي.
2. **تكامل مع أنظمة أخرى**: يمكن دمج البيانات مع أنظمة أخرى تستخدم MongoDB.
3. **نسخ احتياطي متقدم**: استخدام أدوات النسخ الاحتياطي المضمنة في MongoDB مثل `mongodump` و `mongorestore`.
4. **قابلية للتوسع**: يمكن توسيع نطاق MongoDB لاستيعاب كميات كبيرة من البيانات.
5. **أمان إضافي**: يمكن تكوين MongoDB لاستخدام TLS/SSL، والمصادقة، والتشفير.

## الملاحظات

- يتم تخزين جميع بيانات الجلسات في مجموعة `sessions` في قاعدة البيانات المحددة.
- تأكد من أمان قاعدة بيانات MongoDB الخاصة بك، خاصةً إذا كنت تستخدم خادمًا عن بعد.
- يوصى بتمكين المصادقة في MongoDB لمنع الوصول غير المصرح به إلى البيانات. 