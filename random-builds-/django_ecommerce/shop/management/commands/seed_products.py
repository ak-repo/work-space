from django.core.management.base import BaseCommand
from django.utils.text import slugify

from shop.models import Product


PRODUCTS = [
    ('Wireless Headphones', 'Comfortable Bluetooth headphones for everyday use.', '79.99', 20),
    ('Mechanical Keyboard', 'Compact mechanical keyboard with tactile switches.', '99.00', 15),
    ('Smart Watch', 'Simple fitness and notification smart watch.', '129.50', 12),
    ('USB-C Hub', 'Multi-port USB-C hub with HDMI and card reader.', '39.90', 25),
    ('Laptop Stand', 'Adjustable aluminum laptop stand.', '45.00', 18),
    ('Wireless Mouse', 'Ergonomic rechargeable wireless mouse.', '34.99', 30),
]


class Command(BaseCommand):
    help = 'Create sample products for local testing.'

    def handle(self, *args, **options):
        created_count = 0
        for name, description, price, stock in PRODUCTS:
            _, created = Product.objects.update_or_create(
                slug=slugify(name),
                defaults={
                    'name': name,
                    'description': description,
                    'price': price,
                    'stock': stock,
                    'is_active': True,
                },
            )
            created_count += int(created)
        self.stdout.write(self.style.SUCCESS(f'Seed complete. {created_count} new products created.'))
