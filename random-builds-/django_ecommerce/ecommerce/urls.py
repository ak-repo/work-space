from django.contrib import admin
from django.contrib.auth import views as auth_views
from django.urls import include, path

from shop import views as shop_views

urlpatterns = [
    path('admin/', admin.site.urls),
    path('login/', auth_views.LoginView.as_view(template_name='registration/login.html'), name='login'),
    path('logout/', shop_views.logout_view, name='logout'),
    path('register/', shop_views.register, name='register'),
    path('', include('shop.urls')),
]
