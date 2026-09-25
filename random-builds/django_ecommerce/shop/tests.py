from django.contrib.auth.models import User
from django.test import TestCase
from django.urls import reverse

from .models import Order, Payment, Product, WishlistItem


class ShopFlowTests(TestCase):
    def setUp(self):
        self.user = User.objects.create_user(username='tester', password='testpass123')
        self.product = Product.objects.create(
            name='Test Product',
            slug='test-product',
            price='25.00',
            stock=5,
            is_active=True,
        )

    def test_product_list(self):
        response = self.client.get(reverse('shop:product_list'))
        self.assertEqual(response.status_code, 200)
        self.assertContains(response, 'Test Product')

    def test_cart_add(self):
        response = self.client.post(reverse('shop:cart_add', args=[self.product.id]), {'quantity': 2})
        self.assertRedirects(response, reverse('shop:cart_detail'))
        self.assertEqual(self.client.session['cart'][str(self.product.id)], 2)

    def test_wishlist_requires_login_and_adds_product(self):
        self.client.login(username='tester', password='testpass123')
        self.client.post(reverse('shop:wishlist_add', args=[self.product.id]))
        self.assertTrue(WishlistItem.objects.filter(user=self.user, product=self.product).exists())

    def test_checkout_and_mock_payment(self):
        self.client.login(username='tester', password='testpass123')
        self.client.post(reverse('shop:cart_add', args=[self.product.id]), {'quantity': 2})
        response = self.client.post(reverse('shop:checkout'), {
            'shipping_name': 'Test User',
            'shipping_address': '123 Test Street',
        })
        order = Order.objects.get(user=self.user)
        self.assertRedirects(response, reverse('shop:payment', args=[order.id]))
        self.assertEqual(order.total_amount, 50)

        self.client.post(reverse('shop:payment', args=[order.id]))
        order.refresh_from_db()
        self.product.refresh_from_db()
        self.assertEqual(order.payment_status, Order.PaymentStatus.PAID)
        self.assertEqual(order.status, Order.Status.CONFIRMED)
        self.assertEqual(self.product.stock, 3)
        self.assertTrue(Payment.objects.filter(order=order, status=Payment.Status.SUCCESS).exists())
