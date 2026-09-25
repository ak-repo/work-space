import uuid
from decimal import Decimal

from django.contrib import messages
from django.contrib.auth import login, logout
from django.contrib.auth.decorators import login_required
from django.db import transaction
from django.http import HttpResponseBadRequest
from django.shortcuts import get_object_or_404, redirect, render
from django.utils.http import url_has_allowed_host_and_scheme
from django.views.decorators.http import require_POST

from .cart import cart_items, clear_cart, get_cart, remove_product, set_quantity
from .forms import CheckoutForm, RegisterForm
from .models import Order, OrderItem, Payment, Product, WishlistItem


def product_list(request):
    products = Product.objects.filter(is_active=True)
    return render(request, 'shop/product_list.html', {'products': products})


def product_detail(request, slug):
    product = get_object_or_404(Product, slug=slug, is_active=True)
    in_wishlist = False
    if request.user.is_authenticated:
        in_wishlist = WishlistItem.objects.filter(user=request.user, product=product).exists()
    return render(request, 'shop/product_detail.html', {
        'product': product,
        'in_wishlist': in_wishlist,
    })


def register(request):
    if request.user.is_authenticated:
        return redirect('shop:product_list')
    form = RegisterForm(request.POST or None)
    if request.method == 'POST' and form.is_valid():
        user = form.save()
        login(request, user)
        messages.success(request, 'Account created successfully.')
        return redirect('shop:product_list')
    return render(request, 'registration/register.html', {'form': form})


@require_POST
def logout_view(request):
    logout(request)
    messages.success(request, 'Logged out successfully.')
    return redirect('shop:product_list')


@require_POST
def cart_add(request, product_id):
    product = get_object_or_404(Product, id=product_id, is_active=True)
    if product.stock <= 0:
        messages.error(request, 'This product is out of stock.')
        return redirect(product.get_absolute_url())

    try:
        quantity = max(1, int(request.POST.get('quantity', 1)))
    except ValueError:
        quantity = 1

    cart = get_cart(request)
    current_quantity = cart.get(str(product.id), 0)
    set_quantity(request, product.id, min(current_quantity + quantity, product.stock))
    messages.success(request, f'{product.name} added to cart.')
    return redirect('shop:cart_detail')


def cart_detail(request):
    items, total = cart_items(request)
    return render(request, 'shop/cart.html', {'items': items, 'total': total})


@require_POST
def cart_update(request, product_id):
    product = get_object_or_404(Product, id=product_id, is_active=True)
    try:
        quantity = int(request.POST.get('quantity', 1))
    except ValueError:
        return HttpResponseBadRequest('Invalid quantity')

    if quantity > product.stock:
        quantity = product.stock
        messages.warning(request, 'Quantity reduced to available stock.')
    set_quantity(request, product.id, quantity)
    return redirect('shop:cart_detail')


@require_POST
def cart_remove(request, product_id):
    remove_product(request, product_id)
    messages.success(request, 'Item removed from cart.')
    return redirect('shop:cart_detail')


@login_required
def wishlist(request):
    items = WishlistItem.objects.filter(user=request.user).select_related('product')
    return render(request, 'shop/wishlist.html', {'items': items})


@login_required
@require_POST
def wishlist_add(request, product_id):
    product = get_object_or_404(Product, id=product_id, is_active=True)
    WishlistItem.objects.get_or_create(user=request.user, product=product)
    messages.success(request, f'{product.name} added to wishlist.')
    next_url = request.POST.get('next')
    if next_url and url_has_allowed_host_and_scheme(next_url, allowed_hosts={request.get_host()}):
        return redirect(next_url)
    return redirect('shop:wishlist')


@login_required
@require_POST
def wishlist_remove(request, product_id):
    WishlistItem.objects.filter(user=request.user, product_id=product_id).delete()
    messages.success(request, 'Item removed from wishlist.')
    next_url = request.POST.get('next')
    if next_url and url_has_allowed_host_and_scheme(next_url, allowed_hosts={request.get_host()}):
        return redirect(next_url)
    return redirect('shop:wishlist')


@login_required
def checkout(request):
    items, total = cart_items(request)
    if not items:
        messages.warning(request, 'Your cart is empty.')
        return redirect('shop:cart_detail')

    form = CheckoutForm(request.POST or None, initial={'shipping_name': request.user.get_full_name() or request.user.username})
    if request.method == 'POST' and form.is_valid():
        with transaction.atomic():
            locked_products = {
                product.id: product
                for product in Product.objects.select_for_update().filter(
                    id__in=[item['product'].id for item in items],
                    is_active=True,
                )
            }

            validated_items = []
            validated_total = Decimal('0.00')
            for item in items:
                product = locked_products.get(item['product'].id)
                quantity = item['quantity']
                if not product or product.stock < quantity:
                    messages.error(request, f'Insufficient stock for {item["product"].name}.')
                    return redirect('shop:cart_detail')
                line_total = product.price * quantity
                validated_total += line_total
                validated_items.append((product, quantity, line_total))

            order = Order.objects.create(
                user=request.user,
                total_amount=validated_total,
                shipping_name=form.cleaned_data['shipping_name'],
                shipping_address=form.cleaned_data['shipping_address'],
            )

            order_items = []
            for product, quantity, line_total in validated_items:
                product.stock -= quantity
                product.save(update_fields=['stock'])
                order_items.append(OrderItem(
                    order=order,
                    product=product,
                    product_name=product.name,
                    unit_price=product.price,
                    quantity=quantity,
                    line_total=line_total,
                ))
            OrderItem.objects.bulk_create(order_items)

        clear_cart(request)
        messages.success(request, f'Order #{order.id} created. Complete payment to confirm it.')
        return redirect('shop:payment', order_id=order.id)

    return render(request, 'shop/checkout.html', {
        'form': form,
        'items': items,
        'total': total,
    })


@login_required
def order_list(request):
    orders = Order.objects.filter(user=request.user)
    return render(request, 'shop/order_list.html', {'orders': orders})


@login_required
def order_detail(request, order_id):
    order = get_object_or_404(Order.objects.prefetch_related('items'), id=order_id, user=request.user)
    return render(request, 'shop/order_detail.html', {'order': order})


@login_required
def payment(request, order_id):
    order = get_object_or_404(Order, id=order_id, user=request.user)

    if request.method == 'POST':
        if order.payment_status == Order.PaymentStatus.PAID:
            messages.info(request, 'This order is already paid.')
            return redirect('shop:order_detail', order_id=order.id)

        with transaction.atomic():
            order = Order.objects.select_for_update().get(id=order.id, user=request.user)
            if order.payment_status != Order.PaymentStatus.PAID:
                Payment.objects.update_or_create(
                    order=order,
                    defaults={
                        'provider': 'mock',
                        'transaction_id': f'MOCK-{uuid.uuid4().hex[:20].upper()}',
                        'status': Payment.Status.SUCCESS,
                        'amount': order.total_amount,
                    },
                )
                order.payment_status = Order.PaymentStatus.PAID
                order.status = Order.Status.CONFIRMED
                order.save(update_fields=['payment_status', 'status', 'updated_at'])

        messages.success(request, 'Mock payment successful. Order confirmed.')
        return redirect('shop:order_detail', order_id=order.id)

    return render(request, 'shop/payment.html', {'order': order})
