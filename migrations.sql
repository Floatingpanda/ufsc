
-- Table Definition
CREATE TABLE logins (
    id uuid PRIMARY KEY NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tokens (
    id	UUID PRIMARY KEY,
    user_id	UUID REFERENCES users(id),
    name	TEXT,
    value	TEXT,
    counter	int4,
    created_at	timestamptz NOT NULL DEFAULT NOW(),
    updated_at	timestamptz NOT NULL DEFAULT NOW()
);


CREATE TABLE users (
    id UUID PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    password TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'UNCONFIRMED',
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    ssn VARCHAR(12),
    sms_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    active_role TEXT NOT NULL DEFAULT 'STUDENT',

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tutors (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    alias TEXT NOT NULL,
    image TEXT NOT NULL,
    location_id INT REFERENCES locations(id),
    online_lessons BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);



CREATE TABLE tutor_subjects (
    tutor_id	UUID REFERENCES tutors(id),
    subject_id	int REFERENCES subjects(id),
    PRIMARY KEY (tutor_id, subject_id)
);

CREATE TABLE tutor_levels (
    tutor_id	UUID REFERENCES tutors(id),
    level_id	int REFERENCES levels(id),
    PRIMARY KEY (tutor_id, level_id)
);

CREATE TABLE tutor_locations (
    tutor_id	UUID REFERENCES tutors(id),
    location_id	int REFERENCES locations(id),
    PRIMARY KEY (tutor_id, location_id)
);

CREATE TABLE logins (
    id	UUID PRIMARY KEY,
    user_id	UUID REFERENCES users(id),
    created_at	timestamptz NOT NULL DEFAULT NOW(),
    updated_at	timestamptz NOT NULL DEFAULT NOW()
);

--- mails
CREATE TABLE mails (
    id SERIAL PRIMARY KEY,
    "to" TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT
);


CREATE TABLE locations (
    id int SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE subjects (
    id int SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE levels (
    id int SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);


CREATE TABLE lessons (
    id UUID PRIMARY KEY,
    student_id UUID REFERENCES users(id),
    subject_id INT REFERENCES subjects(id),
    level_id INT REFERENCES levels(id),
    location_id INT REFERENCES locations(id),
    online_lesson BOOLEAN NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    start_at TIMESTAMPTZ NOT NULL,
    tutor_id UUID REFERENCES tutors(id),
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lesson_requests (
    id SERIAL PRIMARY KEY,
    lesson_id UUID REFERENCES lessons(id),
    tutor_id UUID REFERENCES tutors(id),
    status TEXT NOT NULL DEFAULT 'PENDING',
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ratings (
    id SERIAL PRIMARY KEY,
    lesson_id UUID REFERENCES lessons(id),
    tutor_id UUID REFERENCES tutors(id),
    grade INT NOT NULL DEFAULT 0,
    feedback TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    lesson_id UUID REFERENCES lessons(id),
    discount_id INT REFERENCES discounts(id),
    quantity INT NOT NULL,
    amount NUMERIC NOT NULL,
    tax_amount NUMERIC NOT NULL,
    discount_amount NUMERIC NOT NULL,
    product_name TEXT NOT NULL,
    product_cost NUMERIC NOT NULL,
    product_tax NUMERIC NOT NULL,
    currency TEXT NOT NULL,
    reference_user TEXT NOT NULL,
    reference_number TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    product_category TEXT NOT NULL
);

CREATE TABLE order_payments (
    id UUID PRIMARY KEY,
    order_id INT REFERENCES orders(id),
    message TEXT NOT NULL,
    reference TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)

CREATE TABLE subscription_payments (
    id UUID PRIMARY KEY,
    subscription_id INT REFERENCES subscriptions(id),
    message TEXT NOT NULL,
    reference TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    plan_id INT REFERENCES plans(id),
    user_id INT REFERENCES users(id),
    discount_id INT REFERENCES discounts(id),
    reference TEXT NOT NULL,
    amount NUMERIC NOT NULL,
    tax_amount NUMERIC NOT NULL,
    discount_amount NUMERIC NOT NULL,
    activated_at TIMESTAMPTZ,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE plans (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    role TEXT,
    period INT NOT NULL,
    interval INT NOT NULL,
    price NUMERIC NOT NULL,
    tax NUMERIC NOT NULL,
    currency TEXT NOT NULL,
    active BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE discounts (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL,
    amount NUMERIC NOT NULL DEFAULT 0,
    valid_to TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    campaign BOOLEAN NOT NULL DEFAULT FALSE,
    is_percent BOOLEAN NOT NULL DEFAULT TRUE,
    currency VARCHAR(3)
);

INSERT INTO LOCATIONS (id,name) values (1,	'Stockholm');
INSERT INTO LOCATIONS (id,name) values (2,	'Göteborg');
INSERT INTO LOCATIONS (id,name) values (3,	'Malmö');
INSERT INTO LOCATIONS (id,name) values (4,	'Blekinge');
INSERT INTO LOCATIONS (id,name) values (5,	'Bohuslän');
INSERT INTO LOCATIONS (id,name) values (6,	'Dalarna');
INSERT INTO LOCATIONS (id,name) values (7,	'Dalsland');
INSERT INTO LOCATIONS (id,name) values (8,	'Gotland');
INSERT INTO LOCATIONS (id,name) values (9,	'Gästrikland');
INSERT INTO LOCATIONS (id,name) values (10,	'Halland');
INSERT INTO LOCATIONS (id,name) values (11,	'Hälsingland');
INSERT INTO LOCATIONS (id,name) values (12,	'Härjedalen');
INSERT INTO LOCATIONS (id,name) values (13,	'Jämtland');
INSERT INTO LOCATIONS (id,name) values (14,	'Lappland');
INSERT INTO LOCATIONS (id,name) values (15,	'Medelpad');
INSERT INTO LOCATIONS (id,name) values (16,	'Norrbotten');
INSERT INTO LOCATIONS (id,name) values (17,	'Närke');
INSERT INTO LOCATIONS (id,name) values (18,	'Skåne');
INSERT INTO LOCATIONS (id,name) values (19,	'Småland');
INSERT INTO LOCATIONS (id,name) values (20,	'Södermanland');
INSERT INTO LOCATIONS (id,name) values (21,	'Uppland');
INSERT INTO LOCATIONS (id,name) values (22,	'Värmland');
INSERT INTO LOCATIONS (id,name) values (23,	'Västerbotten');
INSERT INTO LOCATIONS (id,name) values (24,	'Västergötland');
INSERT INTO LOCATIONS (id,name) values (25,	'Västmanland');
INSERT INTO LOCATIONS (id,name) values (26,	'Ångermanland');
INSERT INTO LOCATIONS (id,name) values (27,	'Öland');
INSERT INTO LOCATIONS (id,name) values (28,	'Östergötland');
INSERT INTO LOCATIONS (id,name) values (29,	'Åland');

INSERT INTO LOCATIONS (id,name) values (-1,'online');
INSERT INTO LOCATIONS (id,name) values (1,'Stockholm');
INSERT INTO LOCATIONS (id,name) values (2,'Göteborg');
INSERT INTO LOCATIONS (id,name) values (3,'Malmö');
INSERT INTO LOCATIONS (id,name) values (4,'Uppsala');
INSERT INTO LOCATIONS (id,name) values (5,'Linköping');
INSERT INTO LOCATIONS (id,name) values (6,'Västerås');
INSERT INTO LOCATIONS (id,name) values (7,'Örebro');
INSERT INTO LOCATIONS (id,name) values (8,'Helsingborg');
INSERT INTO LOCATIONS (id,name) values (9,'Blekinge');
INSERT INTO LOCATIONS (id,name) values (10,'Bohuslän');
INSERT INTO LOCATIONS (id,name) values (11,'Dalarna');
INSERT INTO LOCATIONS (id,name) values (12,'Dalsland');
INSERT INTO LOCATIONS (id,name) values (13,'Gotland');
INSERT INTO LOCATIONS (id,name) values (14,'Gästrikland');
INSERT INTO LOCATIONS (id,name) values (15,'Halland');
INSERT INTO LOCATIONS (id,name) values (16,'Hälsingland');
INSERT INTO LOCATIONS (id,name) values (17,'Härjedalen');
INSERT INTO LOCATIONS (id,name) values (18,'Jämtland');
INSERT INTO LOCATIONS (id,name) values (19,'Lappland');
INSERT INTO LOCATIONS (id,name) values (20,'Medelpad');
INSERT INTO LOCATIONS (id,name) values (21,'Norrbotten');
INSERT INTO LOCATIONS (id,name) values (22,'Närke');
INSERT INTO LOCATIONS (id,name) values (23,'Skåne');
INSERT INTO LOCATIONS (id,name) values (24,'Småland');
INSERT INTO LOCATIONS (id,name) values (25,'Södermanland');
INSERT INTO LOCATIONS (id,name) values (26,'Uppland');
INSERT INTO LOCATIONS (id,name) values (27,'Värmland');
INSERT INTO LOCATIONS (id,name) values (28,'Västerbotten');
INSERT INTO LOCATIONS (id,name) values (29,'Västergötland');
INSERT INTO LOCATIONS (id,name) values (30,'Västmanland');
INSERT INTO LOCATIONS (id,name) values (31,'Ångermanland');
INSERT INTO LOCATIONS (id,name) values (32,'Öland');
INSERT INTO LOCATIONS (id,name) values (33,'Östergötland');
INSERT INTO LOCATIONS (id,name) values (34,'Åland');


INSERT INTO subjects (id,name) values (1,'Biologi');
INSERT INTO subjects (id,name) values (2,'Ekonomi');
INSERT INTO subjects (id,name) values (3,'Engelska');
INSERT INTO subjects (id,name) values (4,'Franska');
INSERT INTO subjects (id,name) values (5,'Fysik');
INSERT INTO subjects (id,name) values (6,'Historia');
INSERT INTO subjects (id,name) values (7,'Italienska');
INSERT INTO subjects (id,name) values (8,'Juridik');
INSERT INTO subjects (id,name) values (9,'Kemi');
INSERT INTO subjects (id,name) values (10,'Matematik');
INSERT INTO subjects (id,name) values (11,'Nationella prov');
INSERT INTO subjects (id,name) values (12,'Programmering');
INSERT INTO subjects (id,name) values (13,'Psykologi');
INSERT INTO subjects (id,name) values (14,'Samhällskunskap');
INSERT INTO subjects (id,name) values (15,'Spanska');
INSERT INTO subjects (id,name) values (16,'Svenska');
INSERT INTO subjects (id,name) values (17,'Tyska');
INSERT INTO subjects (id,name) values (18,'Övriga språk');


INSERT INTO levels (id,name) values (1,'Lågstadiet');
INSERT INTO levels (id,name) values (2,'Mellanstadiet');
INSERT INTO levels (id,name) values (3,'Högstadiet');
INSERT INTO levels (id,name) values (4,'Gymnasiet');
INSERT INTO levels (id,name) values (5,'Universitet/Högskola');
INSERT INTO levels (id,name) values (6,'Vuxenutbildning');