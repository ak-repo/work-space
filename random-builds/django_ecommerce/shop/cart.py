from decimal import Decimal

from .models import Product

CART_SESSION_KEY = 'cart'


def get_cart(request):
    return request.session.get(CART_SESSION_KEY, {})


def save_cart(request, cart):
    request.session[CART_SESSION_KEY] = cart
    request.session.modified = True



def set_quantity(request, product_id, quantity):
    cart = get_cart(request)
    key = str(product_id)
    if quantity <= 0:
        cart.pop(key, None)
    else:
        cart[key] = quantity
    save_cart(request, cart)


def remove_product(request, product_id):
    cart = get_cart(request)
    cart.pop(str(product_id), None)
    save_cart(request, cart)


def clear_cart(request):
    request.session[CART_SESSION_KEY] = {}
    request.session.modified = True


def cart_items(request):
    cart = get_cart(request)
    product_ids = [int(product_id) for product_id in cart.keys()]
    products = Product.objects.filter(id__in=product_ids, is_active=True)
    product_map = {product.id: product for product in products}

    items = []
    total = Decimal('0.00')
    for product_id, quantity in cart.items():
        product = product_map.get(int(product_id))
        if not product:
            continue
        line_total = product.price * quantity
        total += line_total
        items.append({
            'product': product,
            'quantity': quantity,
            'line_total': line_total,
        })
    return items, total
