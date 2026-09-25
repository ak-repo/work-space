# Simple Django E-commerce

A small server-rendered Django project for learning/testing basic e-commerce flows.

## Features

- User registration, login, and logout
- Product list and product detail
- Session-based cart
- User wishlist
- Checkout with shipping name/address
- Order and order-item persistence
- Simple stock validation/deduction
- Mock payment flow (no external payment gateway or API keys)
- Order history and order detail
- Django admin
- Sample product seed command
- Basic Django tests

## Tech

- Python 3.11+
- Django 5.2.x
- SQLite
- Django templates + plain HTML/CSS

## Project layout

```text
simple_django_ecommerce/
├── ecommerce/              # Django project settings/URLs
├── shop/                   # E-commerce application
│   ├── management/commands/seed_products.py
│   ├── migrations/
│   ├── static/shop/css/
│   ├── templates/
│   ├── admin.py
│   ├── cart.py
│   ├── forms.py
│   ├── models.py
│   ├── tests.py
│   ├── urls.py
│   └── views.py
├── manage.py
├── requirements.txt
└── README.md
```

## Local run

### 1. Create virtual environment

Linux/macOS:

```bash
python3 -m venv .venv
source .venv/bin/activate
```

Windows PowerShell:

```powershell
py -m venv .venv
.venv\Scripts\Activate.ps1
```

### 2. Install dependencies

```bash
pip install -r requirements.txt
```

### 3. Create database

```bash
python manage.py migrate
```

### 4. Add sample products

```bash
python manage.py seed_products
```

### 5. Optional: create admin user

```bash
python manage.py createsuperuser
```

### 6. Run server

```bash
python manage.py runserver
```

Open:

- Store: `http://127.0.0.1:8000/`
- Admin: `http://127.0.0.1:8000/admin/`

Register a normal user from `/register/` to test wishlist, checkout, payment, and orders.

## Test flow manually

1. Run `python manage.py seed_products`.
2. Open the product page and add products to cart.
3. Register/login.
4. Add/remove products from wishlist.
5. Open cart and update quantities.
6. Checkout with a shipping name and address.
7. Click **Pay Now (Mock Success)**.
8. Confirm the order shows `Confirmed` and `Paid`.
9. Open **Orders** to review order history.

## Automated tests

Run:

```bash
python manage.py test
```

The tests cover:

- Product list rendering
- Cart add
- Wishlist add
- Checkout/order creation
- Mock payment
- Stock deduction

## Important payment note

`shop.views.payment` is a **mock payment** implementation. It creates a local `Payment` record and marks the order paid/confirmed. It does not charge a card or connect to Stripe/Razorpay/PayPal.

For a real project, replace that view with a payment provider integration and verify payment using provider-side callbacks/webhooks before marking an order as paid.

## Development notes

- `DEBUG=True` and the included secret key are for local development only.
- SQLite is used so the project runs without configuring PostgreSQL/MySQL.
- Cart data is stored in the Django session, so anonymous users can add items before logging in.
