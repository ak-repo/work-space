from .cart import get_cart


def cart_summary(request):
    cart = get_cart(request)
    return {'cart_count': sum(cart.values())}
