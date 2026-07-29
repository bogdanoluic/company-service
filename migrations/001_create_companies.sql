CREATE TABLE companies (
    id UUID PRIMARY KEY,
    name VARCHAR(15) NOT NULL,
    description VARCHAR(3000),
    amount_of_employees INTEGER NOT NULL,
    registered BOOLEAN NOT NULL,
    type TEXT NOT NULL,

    CONSTRAINT companies_name_unique
        UNIQUE (name),

    CONSTRAINT companies_amount_of_employees_non_negative
        CHECK (amount_of_employees >= 0),

    CONSTRAINT companies_type_valid
        CHECK (
            type IN (
                'Corporations',
                'NonProfit',
                'Cooperative',
                'Sole Proprietorship'
            )
        )
);

---- create above / drop below ----

DROP TABLE companies;