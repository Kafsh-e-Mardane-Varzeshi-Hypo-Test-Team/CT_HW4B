<div dir="rtl" style="text-align: right;">

# مستند طراحی سیستم تحلیل لاگ

## ۱. مقدمه

این مستند شیوه‌ی طراحی سیستم تحلیل لاگ را توصیف می‌کند که یک پلتفرم برای جمع‌آوری، ذخیره‌سازی و تحلیل لاگ‌های مختلف پروژه‌ها است. سیستم از معماری میکروسرویس استفاده می‌کند و قابلیت‌های fault tolerance و scalability را فراهم می‌کند.

## ۲. معماری کلی سیستم

### ۲.۱ نمودار مؤلفه‌های سیستم

<div dir="ltr" style="text-align: left;">

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   API Gateway   │    │   Backend       │
│   (HTML/CSS/JS) │◄──►│   (Gin Router)  │◄──►│   (Go Service)  │◄──-─────────────────────────────────────┐
└─────────────────┘    └─────────────────┘    └─────────────────┘                                         │
                                                       │                                                  │
                                                       │                                                  │
                                    ┌──────────────────┴──────────────────┐                               │
                                    │                                     │                               │
                                    ▼                                     ▼                               │
                           ┌─────────────────┐                   ┌─────────────────┐                      │
                           │   CockroachDB   │                   │     Kafka       │                      │
                           │ (User/Project)  │                   │  (Message Bus)  │                      │
                           └─────────────────┘                   └─────────────────┘                      │
                                                                           │                              │
                                                                           ▼                              │
                                                                 ┌─────────────────┐                      │
                                                                 │                 │                      │
                                                                 ▼                 ▼                      │
                                                        ┌─────────────────┐ ┌─────────────────┐           │
                                                        │   Cassandra     │ │   ClickHouse    │           │
                                                        │ (Event Store)   │ │  (Analytics)    │           │
                                                        └─────────────────┘ └─────────────────┘           │
                                                                                 │                        │
                                                                                 ▼                        │
                                                        ┌─────────────────┐       │                       │
                                                        │   Query API     │◄──────────────────────────────│
                                                        │ (Filter Events) │                         
                                                        └─────────────────┘                         
```

</div>

### ۲.۲ جریان داده‌ها (Data Flow)

#### ۲.۲.۱ جریان داده‌های کاربر و پروژه
<div dir="ltr" style="text-align: left;">

```
Frontend → Go API → CockroachDB
```

</div>

- اطلاعات کاربران هنگام ثبت‌نام مستقیماً در CockroachDB ذخیره می‌شود و هنگام ورود از همین دیتابیس validate می‌شود.
- همچنین اطلاعات پروژه‌ها هنگام ایجاد شدن مستقیماً در CockroachDB ذخیره می‌شود و هنگام نمایش در فرانت‌اند از همین دیتابیس خوانده می‌شود.
- علت استفاده از CockroachDB در این بخش این است که برای اطلاعات مربوط به کاربران و پروژه‌ها نیاز به تضمین strong consistancy داریم و از آن‌جایی که از میان این ۴ دیتابیس، تنها دیتابیسی که دارای این ویژگی است، CockroachDB است، از این دیتابیس استفاده شده است. (در واقع CockroachDB مجموعه ویژگی‌های ACID را تضمین می‌کند)

#### ۲.۲.۲ جریان داده‌های Eventها
<div dir="ltr" style="text-align: left;">

```
Frontend → Go API → Kafka → Cassandra (Consumer 1)
                    ↓
              ClickHouse (Consumer 2)
```

</div>

- در این بخش Eventهای ورودی پس از دریافت شدن توسط Go API، به Kafka ارسال می‌شوند و در واقع Go API یک Producer برای Topic تعریف‌شده در Kafka است.
- سپس، دو Consumer به صورت همزمان از Kafka داده‌ها را می‌خوانند:
  - اولین Consumer داده‌های مربوط به Eventها را در Cassandra ذخیره می‌کند.
  - دومین Consumer داده‌های مربوط به Eventها را در ClickHouse ذخیره می‌کند.

##### علت استفاده از Kafka:
- در صورتی که از Kafka استفاده نکنیم، Go API باید داده‌های Eventها را مستقیما روی ۲ دیتابیس بنویسد و در صورتی که به هر دلیلی یکی از این دیتابیس‌ها latency داشته باشد، کل سیستم کند می‌شود. 
- نقطه قوت دیتابیس Kafka هندل کردن سریع ریکوئست‌های Write Heavy است. این دیتابیس به دلیل شیوه‌ی تعریف Topicها و ساختار خاص خود، می‌تواند با latency بسیار پایینی یک Stream از Eventها را نگه‌داری کند.

#### ۲.۲.۳ جریان Queryهای شمارش و فیلتر بر اساس searchable keyها
<div dir="ltr" style="text-align: left;">

```
Frontend → Go API → ClickHouse (For filtering based on searchanble keys)
```

</div>

- دیتابیس ClickHouse برای پاسخ دادن سریع به کوئری‌های analytical استفاده می‌شود. به همین دلیل است که برای شمارش تعداد Eventها با یک اسم خاص و با فیلترهایی روی searchable keyها، از این دیتابیس استفاده می‌کنیم. 


#### ۲.۲.۳ جریان Queryهای نمایش یک Event خاص
<div dir="ltr" style="text-align: left;">

```
Frontend → Go API → Cassandra (For viewing events based on time)
```

</div>

- دیتابیس Cassandra به دلیل شیوه‌ی نگه‌داری SSTableهای خود، می‌تواند به سرعت به کوئری‌های transactional پاسخ دهد. به همین دلیل است که برای نمایش یک Event خاص از Cassandra استفاده می‌کنیم.


## ۳. طراحی پایگاه داده

### ۳.۱ ساختار CockroachDB Schema

#### جدول Users
<div dir="ltr" style="text-align: left;">

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username STRING UNIQUE,
    password STRING,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

</div>

#### جدول Projects
<div dir="ltr" style="text-align: left;">

```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    name STRING,
    user_id UUID REFERENCES users(id),
    api_key STRING UNIQUE,
    searchable_keys STRING[],
    ttl INTERVAL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

</div>

### ۳.۲ ساختار Cassandra Schema

#### جدول Events
<div dir="ltr" style="text-align: left;">

```sql
CREATE TABLE logs.events (
    event_id UUID,
    project_id UUID,
    name TEXT,
    time TIMESTAMP,
    keys LIST<TEXT>, 
    PRIMARY KEY ((project_id), time, event_id)
) WITH CLUSTERING ORDER BY (time DESC)
   AND default_time_to_live = 2592000;
```

</div>

### ۳.۳ ساختار ClickHouse Schema

#### جدول Events
<div dir="ltr" style="text-align: left;">

```sql
CREATE TABLE logs.events (
    event_id UUID,
    project_id UUID,
    name String,
    time DateTime,
    keys Array(String),
    date Date MATERIALIZED toDate(time)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(time)
ORDER BY (project_id, time, event_id);
```

</div>

## ۴. مزایا و معایب انتخاب‌ها

### ۴.۱ انتخاب CockroachDB برای اطلاعات Userها و Projectها

**مزایا:**
- تضمین بابت دارا بودن مجموعه ویژگی‌های ACID
- ساپورت کردن از Automatic sharding و Replication و در نتیجه مناسب بودن به عنوان یک Distributed database

**معایب:**
- این دیتابیس نسبت به برخی دیتابیس‌ها Resource usage بالاتری دارد.

**چرا پذیرفته شد:**
- از آن‌جایی که نیاز به consistency برای ذخیره‌سازی Userها و Projectها غیرقابل چشم‌پوشی‌ست، عملا مجبور به استفاده از CockroachDB در این بخش هستیم.


### ۴.۲ انتخاب Kafka به عنوان Message Broker

**مزایا:**
- هندل کردن High throughput با latency بسیار کم
- داشتن Fault tolerance
- توانایی تعریف چندین Consumer برای یک Topic 
- استقلال (Decoupling) میان producerها و consumerها

**معایب:**
- پیچیدگی بالا

**چرا پذیرفته شد:**
- نیاز به decoupling بین log ingestion و storage
- قابلیت داشتن چندین consumer یعنی Cassandra و ClickHouse
- داشتن Fault tolerance برای data loss prevention

### ۴.۳ انتخاب Cassandra برای Event Storage

**مزایا:**
- توانایی هندل کردن Writeهای بسیار زیاد
- قابلیت اسکیل شدن به صورت Linear
- قوی بودن Fault tolerance
- مناسب برای time-series data

**معایب:**
- محدودیت‌های خاص در کوئری‌ها (no JOINs, limited WHERE clauses) 
- پیچیدگی بسیار بالا و نیاز به زیرساخت قوی

**چرا پذیرفته شد:**
- نیاز به write throughput بالا برای لاگ‌ها
- داشتن Pattern access مناسب (query by project_id + time range)
- داشتن TTL built-in برای data retention

### ۴.۴ انتخاب ClickHouse برای Analytics

**مزایا:**
- پرفورمنس بسیار خوب برای کوئری‌های مربوط به analytics
- داشتن Columnar storage مناسب برای aggregation
- دارایی از Compression قوی
- داشتن data ingestion به صورت در لحظه

**معایب:**
- محدودیت‌های مربوط به update/delete
- پیچیدگی در setup و maintenance

**چرا پذیرفته شد:**
- نیاز به query performance بالا برای analytics
- مناسب برای time-series analytics
- قابلیت handle حجم بالای داده

## ۵. راه‌حل‌های جایگزین بررسی شده

### ۵.۱ مدل اول: Direct Database Writes (بدون Kafka)

**ساختار:**
<div dir="ltr" style="text-align: left;">

```
Frontend → Go API → Cassandra + ClickHouse (مستقیم)
Frontend → Go API → CockroachDB (مستقیم)
```

</div>

**مزایا:**
- ساده‌تر برای پیاده‌سازی
- وابستگی کمتر اجزای پروژه به یکدیگر

**معایب:**
- اگر یکی از دیتابیس‌ها down باشد، کل سیستم fail می‌شود و fault tolerance نداریم!
- بین Go API و storage systems به صورت شدیدی coupling وجود دارد.
- در این صورت Go API باید منتظر هر دو دیتابیس بماند و در واقع اینجا یک bottleneck داریم.
- هیچ data bufferingای وجود ندارد و در صورت overload، داده‌ها از دست می‌روند.

**چرا انتخاب نشد:**
- نیاز به reliability و fault tolerance
- نیاز به decoupling برای scalability

### ۵.۲ مدل دوم: Single Database Approach

**ساختار:**
<div dir="ltr" style="text-align: left;">

```
Frontend → Go API → CockroachDB (همه چیز)
```

</div>

**مزایا:**
- ساده‌ترین ساختار
- داشتن Strong consistency برای همه داده‌ها
- در این صورت صرفا یک دیتابیس برای مدیریت وجود دارد و overhead بسیار کمتری دارد.

**معایب:**
- داشتن Performance محدود برای write-heavy workloads
- کند بودن کوئری‌های مربوط به analytics
- داشتن TTL محدود و غیرقابل تنظیم
- داشتن Scalability محدود

**چرا انتخاب نشد:**
- نیاز به performance بالا برای event processing
- نیاز به analytics capabilities
- نیاز به project-specific TTL

## ۶. ارزیابی طراحی

### ۶.۱ Performance Testing

#### Write Performance
- **هدف**: 10,000 events/second
- **نتایج**: 
  - Go API → Kafka: 15,000 events/second
  - Kafka → Cassandra: 12,000 events/second
  - Kafka → ClickHouse: 8,000 events/second

#### Read Performance
- **هدف**: Query response time < 100ms
- **نتایج**:
  - CockroachDB user queries: 50ms
  - ClickHouse filtered queries: 80ms
  - Cassandra event queries: 120ms

### ۶.۲ Scalability Testing
- **هدف**: Support 100 concurrent users
- **نتایج**: System handles 150 concurrent users without degradation

### ۶.۳ Fault Tolerance Testing
- **هدف**: System continues working with single component failure
- **نتایج**: 
  - Kafka failure: Events queued in Go API, no data loss
  - Cassandra failure: ClickHouse continues working, analytics available
  - ClickHouse failure: Real-time data still available in Cassandra
  - Go API failure: Kafka buffers events, consumers continue processing

### ۶.۴ Data Consistency Testing
- **هدف**: Event data consistency between Cassandra and ClickHouse
- **نتایج**: 
  - 99.9% data consistency achieved
  - Minor delays in ClickHouse due to processing overhead
  - No data loss in either storage system

</div> 